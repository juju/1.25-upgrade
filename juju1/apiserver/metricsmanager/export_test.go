// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package metricsmanager

import (
	"github.com/juju/1.25-upgrade/juju1/apiserver/metricsender"
)

func PatchSender(s metricsender.MetricSender) {
	sender = s
}
