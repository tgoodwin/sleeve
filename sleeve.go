package sleeve

import (
	"github.com/tgoodwin/sleeve/client"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func Wrap(wrapped kclient.Client) *client.Client {
	return client.Wrap(wrapped)
}
