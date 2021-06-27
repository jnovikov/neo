package rendezvous

import "math"

const maxUInt64 = float64(^uint64(0))

func combineKey(nodeID, value string) string {
	return nodeID + ":" + value
}

func weightHash(hash uint64, weight int) float64 {
	scale := 1.0 / -math.Log(float64(hash)/maxUInt64)
	return scale * float64(weight)
}
