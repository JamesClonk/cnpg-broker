package cnpg

import (
	"fmt"
)

func clusterName(instanceId string) string {
	return fmt.Sprintf("db-%s", instanceId)
}
