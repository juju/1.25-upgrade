// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// The controller package defines an API end point for functions dealing
// with controllers as a whole.
package controller

import (
	"encoding/json"
	"sort"

	"github.com/juju/errors"
	"github.com/juju/loggo"
	"github.com/juju/txn"
	"github.com/juju/utils/set"
	"gopkg.in/juju/names.v2"
	"gopkg.in/macaroon.v1"

	"github.com/juju/1.25-upgrade/juju2/api"
	"github.com/juju/1.25-upgrade/juju2/api/migrationtarget"
	"github.com/juju/1.25-upgrade/juju2/apiserver/common"
	"github.com/juju/1.25-upgrade/juju2/apiserver/common/cloudspec"
	"github.com/juju/1.25-upgrade/juju2/apiserver/facade"
	"github.com/juju/1.25-upgrade/juju2/apiserver/params"
	coremigration "github.com/juju/1.25-upgrade/juju2/core/migration"
	"github.com/juju/1.25-upgrade/juju2/migration"
	"github.com/juju/1.25-upgrade/juju2/permission"
	"github.com/juju/1.25-upgrade/juju2/state"
	"github.com/juju/1.25-upgrade/juju2/state/stateenvirons"
)

var logger = loggo.GetLogger("juju.apiserver.controller")

// Controller defines the methods on the controller API end point.
type Controller interface {
	AllModels() (params.UserModelList, error)
	DestroyController(args params.DestroyControllerArgs) error
	ModelConfig() (params.ModelConfigResults, error)
	HostedModelConfigs() (params.HostedModelConfigsResults, error)
	GetControllerAccess(params.Entities) (params.UserAccessResults, error)
	ControllerConfig() (params.ControllerConfigResult, error)
	ListBlockedModels() (params.ModelBlockInfoList, error)
	RemoveBlocks(args params.RemoveBlocksArgs) error
	WatchAllModels() (params.AllWatcherId, error)
	ModelStatus(params.Entities) (params.ModelStatusResults, error)
	InitiateMigration(params.InitiateMigrationArgs) (params.InitiateMigrationResults, error)
	ModifyControllerAccess(params.ModifyControllerAccessRequest) (params.ErrorResults, error)
}

// ControllerAPI implements the environment manager interface and is
// the concrete implementation of the api end point.
type ControllerAPI struct {
	*common.ControllerConfigAPI
	*common.ModelStatusAPI
	cloudspec.CloudSpecAPI

	state      *state.State
	statePool  *state.StatePool
	authorizer facade.Authorizer
	apiUser    names.UserTag
	resources  facade.Resources
}

var _ Controller = (*ControllerAPI)(nil)

// NewControllerAPI creates a new api server endpoint for managing
// environments.
func NewControllerAPI(ctx facade.Context) (*ControllerAPI, error) {
	authorizer := ctx.Auth()
	if !authorizer.AuthClient() {
		return nil, errors.Trace(common.ErrPerm)
	}

	// Since we know this is a user tag (because AuthClient is true),
	// we just do the type assertion to the UserTag.
	apiUser, _ := authorizer.GetAuthTag().(names.UserTag)

	st := ctx.State()
	environConfigGetter := stateenvirons.EnvironConfigGetter{st}
	return &ControllerAPI{
		ControllerConfigAPI: common.NewControllerConfig(st),
		ModelStatusAPI: common.NewModelStatusAPI(
			common.NewModelManagerBackend(st),
			common.NewBackendPool(ctx.StatePool()),
			authorizer,
			apiUser,
		),
		CloudSpecAPI: cloudspec.NewCloudSpec(environConfigGetter.CloudSpec, common.AuthFuncForTag(st.ModelTag())),
		state:        st,
		statePool:    ctx.StatePool(),
		authorizer:   authorizer,
		apiUser:      apiUser,
		resources:    ctx.Resources(),
	}, nil
}

func (s *ControllerAPI) checkHasAdmin() error {
	isAdmin, err := s.authorizer.HasPermission(permission.SuperuserAccess, s.state.ControllerTag())
	if err != nil {
		return errors.Trace(err)
	}
	if !isAdmin {
		return common.ServerError(common.ErrPerm)
	}
	return nil
}

// AllModels allows controller administrators to get the list of all the
// models in the controller.
func (s *ControllerAPI) AllModels() (params.UserModelList, error) {
	result := params.UserModelList{}
	if err := s.checkHasAdmin(); err != nil {
		return result, errors.Trace(err)
	}

	// Get all the models that the authenticated user can see, and
	// supplement that with the other models that exist that the user
	// cannot see. The reason we do this is to get the LastConnection
	// time for the models that the user is able to see, so we have
	// consistent output when listing with or without --all when an
	// admin user.
	models, err := s.state.ModelsForUser(s.apiUser)
	if err != nil {
		return result, errors.Trace(err)
	}
	visibleModels := set.NewStrings()
	for _, model := range models {
		lastConn, err := model.LastConnection()
		if err != nil && !state.IsNeverConnectedError(err) {
			return result, errors.Trace(err)
		}
		visibleModels.Add(model.UUID())
		result.UserModels = append(result.UserModels, params.UserModel{
			Model: params.Model{
				Name:     model.Name(),
				UUID:     model.UUID(),
				OwnerTag: model.Owner().String(),
			},
			LastConnection: &lastConn,
		})
	}

	allModels, err := s.state.AllModels()
	if err != nil {
		return result, errors.Trace(err)
	}

	for _, model := range allModels {
		if !visibleModels.Contains(model.UUID()) {
			result.UserModels = append(result.UserModels, params.UserModel{
				Model: params.Model{
					Name:     model.Name(),
					UUID:     model.UUID(),
					OwnerTag: model.Owner().String(),
				},
				// No LastConnection as this user hasn't.
			})
		}
	}

	// Sort the resulting sequence by environment name, then owner.
	sort.Sort(orderedUserModels(result.UserModels))

	return result, nil
}

// ListBlockedModels returns a list of all environments on the controller
// which have a block in place.  The resulting slice is sorted by environment
// name, then owner. Callers must be controller administrators to retrieve the
// list.
func (s *ControllerAPI) ListBlockedModels() (params.ModelBlockInfoList, error) {
	results := params.ModelBlockInfoList{}
	if err := s.checkHasAdmin(); err != nil {
		return results, errors.Trace(err)
	}
	blocks, err := s.state.AllBlocksForController()
	if err != nil {
		return results, errors.Trace(err)
	}

	envBlocks := make(map[string][]string)
	for _, block := range blocks {
		uuid := block.ModelUUID()
		types, ok := envBlocks[uuid]
		if !ok {
			types = []string{block.Type().String()}
		} else {
			types = append(types, block.Type().String())
		}
		envBlocks[uuid] = types
	}

	for uuid, blocks := range envBlocks {
		envInfo, err := s.state.GetModel(names.NewModelTag(uuid))
		if err != nil {
			logger.Debugf("Unable to get name for model: %s", uuid)
			continue
		}
		results.Models = append(results.Models, params.ModelBlockInfo{
			UUID:     envInfo.UUID(),
			Name:     envInfo.Name(),
			OwnerTag: envInfo.Owner().String(),
			Blocks:   blocks,
		})
	}

	// Sort the resulting sequence by environment name, then owner.
	sort.Sort(orderedBlockInfo(results.Models))

	return results, nil
}

// ModelConfig returns the environment config for the controller
// environment.  For information on the current environment, use
// client.ModelGet
func (s *ControllerAPI) ModelConfig() (params.ModelConfigResults, error) {
	result := params.ModelConfigResults{}
	if err := s.checkHasAdmin(); err != nil {
		return result, errors.Trace(err)
	}

	controllerModel, err := s.state.ControllerModel()
	if err != nil {
		return result, errors.Trace(err)
	}

	cfg, err := controllerModel.Config()
	if err != nil {
		return result, errors.Trace(err)
	}

	result.Config = make(map[string]params.ConfigValue)
	for name, val := range cfg.AllAttrs() {
		result.Config[name] = params.ConfigValue{
			Value: val,
		}
	}
	return result, nil
}

// HostedModelConfigs returns all the information that the client needs in
// order to connect directly with the host model's provider and destroy it
// directly.
func (s *ControllerAPI) HostedModelConfigs() (params.HostedModelConfigsResults, error) {
	result := params.HostedModelConfigsResults{}
	if err := s.checkHasAdmin(); err != nil {
		return result, errors.Trace(err)
	}

	controllerModel, err := s.state.ControllerModel()
	if err != nil {
		return result, errors.Trace(err)
	}

	allModels, err := s.state.AllModels()
	if err != nil {
		return result, errors.Trace(err)
	}

	for _, model := range allModels {
		if model.UUID() != controllerModel.UUID() {
			config := params.HostedModelConfig{
				Name:     model.Name(),
				OwnerTag: model.Owner().String(),
			}
			modelConf, err := model.Config()
			if err != nil {
				config.Error = common.ServerError(err)
			} else {
				config.Config = modelConf.AllAttrs()
			}
			cloudSpec := s.GetCloudSpec(model.ModelTag())
			if config.Error == nil {
				config.CloudSpec = cloudSpec.Result
				config.Error = cloudSpec.Error
			}
			result.Models = append(result.Models, config)
		}
	}

	return result, nil
}

// RemoveBlocks removes all the blocks in the controller.
func (s *ControllerAPI) RemoveBlocks(args params.RemoveBlocksArgs) error {
	if err := s.checkHasAdmin(); err != nil {
		return errors.Trace(err)
	}

	if !args.All {
		return errors.New("not supported")
	}
	return errors.Trace(s.state.RemoveAllBlocksForController())
}

// WatchAllModels starts watching events for all models in the
// controller. The returned AllWatcherId should be used with Next on the
// AllModelWatcher endpoint to receive deltas.
func (c *ControllerAPI) WatchAllModels() (params.AllWatcherId, error) {
	if err := c.checkHasAdmin(); err != nil {
		return params.AllWatcherId{}, errors.Trace(err)
	}
	w := c.state.WatchAllModels(c.statePool)
	return params.AllWatcherId{
		AllWatcherId: c.resources.Register(w),
	}, nil
}

type orderedBlockInfo []params.ModelBlockInfo

func (o orderedBlockInfo) Len() int {
	return len(o)
}

func (o orderedBlockInfo) Less(i, j int) bool {
	if o[i].Name < o[j].Name {
		return true
	}
	if o[i].Name > o[j].Name {
		return false
	}

	if o[i].OwnerTag < o[j].OwnerTag {
		return true
	}
	if o[i].OwnerTag > o[j].OwnerTag {
		return false
	}

	// Unreachable based on the rules of there not being duplicate
	// environments of the same name for the same owner, but return false
	// instead of panicing.
	return false
}

// GetControllerAccess returns the level of access the specifed users
// have on the controller.
func (c *ControllerAPI) GetControllerAccess(req params.Entities) (params.UserAccessResults, error) {
	results := params.UserAccessResults{}
	isAdmin, err := c.authorizer.HasPermission(permission.SuperuserAccess, c.state.ControllerTag())
	if err != nil {
		return results, errors.Trace(err)
	}

	users := req.Entities
	results.Results = make([]params.UserAccessResult, len(users))
	for i, user := range users {
		userTag, err := names.ParseUserTag(user.Tag)
		if err != nil {
			results.Results[i].Error = common.ServerError(err)
			continue
		}
		if !isAdmin && !c.authorizer.AuthOwner(userTag) {
			results.Results[i].Error = common.ServerError(common.ErrPerm)
			continue
		}
		access, err := c.state.UserPermission(userTag, c.state.ControllerTag())
		if err != nil {
			results.Results[i].Error = common.ServerError(err)
			continue
		}
		results.Results[i].Result = &params.UserAccess{
			Access:  string(access),
			UserTag: userTag.String()}
	}
	return results, nil
}

// InitiateMigration attempts to begin the migration of one or
// more models to other controllers.
func (c *ControllerAPI) InitiateMigration(reqArgs params.InitiateMigrationArgs) (
	params.InitiateMigrationResults, error,
) {
	out := params.InitiateMigrationResults{
		Results: make([]params.InitiateMigrationResult, len(reqArgs.Specs)),
	}
	if err := c.checkHasAdmin(); err != nil {
		return out, errors.Trace(err)
	}

	for i, spec := range reqArgs.Specs {
		result := &out.Results[i]
		result.ModelTag = spec.ModelTag
		id, err := c.initiateOneMigration(spec)
		if err != nil {
			result.Error = common.ServerError(err)
		} else {
			result.MigrationId = id
		}
	}
	return out, nil
}

func (c *ControllerAPI) initiateOneMigration(spec params.MigrationSpec) (string, error) {
	modelTag, err := names.ParseModelTag(spec.ModelTag)
	if err != nil {
		return "", errors.Annotate(err, "model tag")
	}

	// Ensure the model exists.
	if _, err := c.state.GetModel(modelTag); err != nil {
		return "", errors.Annotate(err, "unable to read model")
	}

	hostedState, err := c.state.ForModel(modelTag)
	if err != nil {
		return "", errors.Trace(err)
	}
	defer hostedState.Close()

	// Construct target info.
	specTarget := spec.TargetInfo
	controllerTag, err := names.ParseControllerTag(specTarget.ControllerTag)
	if err != nil {
		return "", errors.Annotate(err, "controller tag")
	}
	authTag, err := names.ParseUserTag(specTarget.AuthTag)
	if err != nil {
		return "", errors.Annotate(err, "auth tag")
	}
	var macs []macaroon.Slice
	if specTarget.Macaroons != "" {
		if err := json.Unmarshal([]byte(specTarget.Macaroons), &macs); err != nil {
			return "", errors.Annotate(err, "invalid macaroons")
		}
	}
	targetInfo := coremigration.TargetInfo{
		ControllerTag: controllerTag,
		Addrs:         specTarget.Addrs,
		CACert:        specTarget.CACert,
		AuthTag:       authTag,
		Password:      specTarget.Password,
		Macaroons:     macs,
	}

	// Check if the migration is likely to succeed.
	if err := runMigrationPrechecks(hostedState, &targetInfo); err != nil {
		return "", errors.Trace(err)
	}

	// Trigger the migration.
	mig, err := hostedState.CreateMigration(state.MigrationSpec{
		InitiatedBy: c.apiUser,
		TargetInfo:  targetInfo,
	})
	if err != nil {
		return "", errors.Trace(err)
	}
	return mig.Id(), nil
}

// ModifyControllerAccess changes the model access granted to users.
func (c *ControllerAPI) ModifyControllerAccess(args params.ModifyControllerAccessRequest) (params.ErrorResults, error) {
	result := params.ErrorResults{
		Results: make([]params.ErrorResult, len(args.Changes)),
	}
	if len(args.Changes) == 0 {
		return result, nil
	}

	hasPermission, err := c.authorizer.HasPermission(permission.SuperuserAccess, c.state.ControllerTag())
	if err != nil {
		return result, errors.Trace(err)
	}

	for i, arg := range args.Changes {
		if !hasPermission {
			result.Results[i].Error = common.ServerError(common.ErrPerm)
			continue
		}

		controllerAccess := permission.Access(arg.Access)
		if err := permission.ValidateControllerAccess(controllerAccess); err != nil {
			result.Results[i].Error = common.ServerError(err)
			continue
		}

		targetUserTag, err := names.ParseUserTag(arg.UserTag)
		if err != nil {
			result.Results[i].Error = common.ServerError(errors.Annotate(err, "could not modify controller access"))
			continue
		}

		result.Results[i].Error = common.ServerError(
			ChangeControllerAccess(c.state, c.apiUser, targetUserTag, arg.Action, controllerAccess))
	}
	return result, nil
}

// runMigrationPrechecks runs prechecks on the migration and updates
// information in targetInfo as needed based on information
// retrieved from the target controller.
var runMigrationPrechecks = func(st *state.State, targetInfo *coremigration.TargetInfo) error {
	// Check model and source controller.
	backend, err := migration.PrecheckShim(st)
	if err != nil {
		return errors.Annotate(err, "creating backend")
	}
	if err := migration.SourcePrecheck(backend); err != nil {
		return errors.Annotate(err, "source prechecks failed")
	}

	// Check target controller.
	conn, err := api.Open(targetToAPIInfo(targetInfo), migration.ControllerDialOpts())
	if err != nil {
		return errors.Annotate(err, "connect to target controller")
	}
	defer conn.Close()
	modelInfo, err := makeModelInfo(st)
	if err != nil {
		return errors.Trace(err)
	}
	client := migrationtarget.NewClient(conn)
	if targetInfo.CACert == "" {
		targetInfo.CACert, err = client.CACert()
		if err != nil {
			if !params.IsCodeNotImplemented(err) {
				return errors.Annotatef(err, "cannot retrieve CA certificate")
			}
			// If the call's not implemented, it indicates an earlier version
			// of the controller, which we can't migrate to.
			return errors.New("controller API version is too old")
		}
	}
	err = client.Prechecks(modelInfo)
	return errors.Annotate(err, "target prechecks failed")
}

func makeModelInfo(st *state.State) (coremigration.ModelInfo, error) {
	var empty coremigration.ModelInfo

	model, err := st.Model()
	if err != nil {
		return empty, errors.Trace(err)
	}

	// Retrieve agent version for the model.
	conf, err := st.ModelConfig()
	if err != nil {
		return empty, errors.Trace(err)
	}
	agentVersion, _ := conf.AgentVersion()

	// Retrieve agent version for the controller.
	controllerModel, err := st.ControllerModel()
	if err != nil {
		return empty, errors.Trace(err)
	}
	controllerConfig, err := controllerModel.Config()
	if err != nil {
		return empty, errors.Trace(err)
	}
	controllerVersion, _ := controllerConfig.AgentVersion()

	return coremigration.ModelInfo{
		UUID:                   model.UUID(),
		Name:                   model.Name(),
		Owner:                  model.Owner(),
		AgentVersion:           agentVersion,
		ControllerAgentVersion: controllerVersion,
	}, nil
}

func targetToAPIInfo(ti *coremigration.TargetInfo) *api.Info {
	return &api.Info{
		Addrs:     ti.Addrs,
		CACert:    ti.CACert,
		Tag:       ti.AuthTag,
		Password:  ti.Password,
		Macaroons: ti.Macaroons,
	}
}

func grantControllerAccess(accessor *state.State, targetUserTag, apiUser names.UserTag, access permission.Access) error {
	_, err := accessor.AddControllerUser(state.UserAccessSpec{User: targetUserTag, CreatedBy: apiUser, Access: access})
	if errors.IsAlreadyExists(err) {
		controllerTag := accessor.ControllerTag()
		controllerUser, err := accessor.UserAccess(targetUserTag, controllerTag)
		if errors.IsNotFound(err) {
			// Conflicts with prior check, must be inconsistent state.
			err = txn.ErrExcessiveContention
		}
		if err != nil {
			return errors.Annotate(err, "could not look up controller access for user")
		}

		// Only set access if greater access is being granted.
		if controllerUser.Access.EqualOrGreaterControllerAccessThan(access) {
			return errors.Errorf("user already has %q access or greater", access)
		}
		if _, err = accessor.SetUserAccess(controllerUser.UserTag, controllerUser.Object, access); err != nil {
			return errors.Annotate(err, "could not set controller access for user")
		}
		return nil

	}
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

func revokeControllerAccess(accessor *state.State, targetUserTag, apiUser names.UserTag, access permission.Access) error {
	controllerTag := accessor.ControllerTag()
	switch access {
	case permission.LoginAccess:
		// Revoking login access removes all access.
		err := accessor.RemoveUserAccess(targetUserTag, controllerTag)
		return errors.Annotate(err, "could not revoke controller access")
	case permission.AddModelAccess:
		// Revoking add-model access sets login.
		controllerUser, err := accessor.UserAccess(targetUserTag, controllerTag)
		if err != nil {
			return errors.Annotate(err, "could not look up controller access for user")
		}
		_, err = accessor.SetUserAccess(controllerUser.UserTag, controllerUser.Object, permission.LoginAccess)
		return errors.Annotate(err, "could not set controller access to read-only")
	case permission.SuperuserAccess:
		// Revoking superuser sets add-model.
		controllerUser, err := accessor.UserAccess(targetUserTag, controllerTag)
		if err != nil {
			return errors.Annotate(err, "could not look up controller access for user")
		}
		_, err = accessor.SetUserAccess(controllerUser.UserTag, controllerUser.Object, permission.AddModelAccess)
		return errors.Annotate(err, "could not set controller access to add-model")

	default:
		return errors.Errorf("don't know how to revoke %q access", access)
	}

}

// ChangeControllerAccess performs the requested access grant or revoke action for the
// specified user on the controller.
func ChangeControllerAccess(accessor *state.State, apiUser, targetUserTag names.UserTag, action params.ControllerAction, access permission.Access) error {
	switch action {
	case params.GrantControllerAccess:
		err := grantControllerAccess(accessor, targetUserTag, apiUser, access)
		if err != nil {
			return errors.Annotate(err, "could not grant controller access")
		}
		return nil
	case params.RevokeControllerAccess:
		return revokeControllerAccess(accessor, targetUserTag, apiUser, access)
	default:
		return errors.Errorf("unknown action %q", action)
	}
}

func (o orderedBlockInfo) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}

type orderedUserModels []params.UserModel

func (o orderedUserModels) Len() int {
	return len(o)
}

func (o orderedUserModels) Less(i, j int) bool {
	if o[i].Name < o[j].Name {
		return true
	}
	if o[i].Name > o[j].Name {
		return false
	}

	if o[i].OwnerTag < o[j].OwnerTag {
		return true
	}
	if o[i].OwnerTag > o[j].OwnerTag {
		return false
	}

	// Unreachable based on the rules of there not being duplicate
	// environments of the same name for the same owner, but return false
	// instead of panicing.
	return false
}

func (o orderedUserModels) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}
