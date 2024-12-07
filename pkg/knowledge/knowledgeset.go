package knowledge

import (
	"github.com/tgoodwin/sleeve/pkg/event"
	"github.com/tgoodwin/sleeve/pkg/snapshot"
	"github.com/tgoodwin/sleeve/pkg/util"
)

type VersionKnowlege util.Set[snapshot.VersionKey]
type ObjectVersions util.Set[event.CausalKey]
