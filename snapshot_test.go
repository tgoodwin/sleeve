package client

// These are just examples I was working thropugh to understand how the code in this module can be used to reason about a few bugs.
// TODO actually write unit tests for these functions
// - @tgoodwin

// Zookeeper-314
// https://github.com/pravega/zookeeper-operator/issues/314
//
//
//	rule: {
//		scope: kind, uid
//			versions ordered by observation time should be non-decreasing
//	}
//  violation: {
//		kind: ZookeeperCluster
//		uid: 001
//		interval: [t7,]
//	}
//
// actual fix:
// change controller List operation to always do a quorum read against etcd upon restart

var zk = LocalKnowledge{
	{
		// learns of a new ZookeeperCluster, issues command to create a StatefulSet for it
		timestamp: "t1", elements: VersionSet{
			{Kind: "ZookeeperCluster", Uid: "001", Version: "43"}: {}, // scale=2
		},
	},
	{ // observes effect of the action taken at t1
		timestamp: "t2", elements: VersionSet{
			{Kind: "ZookeeperCluster", Uid: "001", Version: "43"}: {},
			{Kind: "StatefulSet", Uid: "002", Version: "49"}:      {},
		},
	},
	{
		// learns of the scaledown request for the zookeeper cluster 001
		// will then issue a command to scale down the StatefulSet
		timestamp: "t3", elements: VersionSet{
			{Kind: "ZookeeperCluster", Uid: "001", Version: "210"}: {}, // scale=2 -> scale=1
			{Kind: "StatefulSet", Uid: "002", Version: "49"}:       {},
		},
	},
	{
		// observes effect of the action taken at t3
		timestamp: "t4", elements: VersionSet{
			{Kind: "ZookeeperCluster", Uid: "001", Version: "210"}: {}, // scale=1
			{Kind: "StatefulSet", Uid: "002", Version: "240"}:      {}, // scale=1
		},
	},
	// HERE, APISERVER 2 GETS PARTITIONED
	{
		// now, the controller learns of the scaleup request for the zookeeper cluster 001
		// will then issue a command to scale up the StatefulSet
		timestamp: "t5", elements: VersionSet{
			{Kind: "ZookeeperCluster", Uid: "001", Version: "500"}: {}, // scale=1 -> scale=2
			{Kind: "StatefulSet", Uid: "002", Version: "240"}:      {}, // scale=1
		},
	},
	{
		// observes effect of the action taken at t5
		timestamp: "t6", elements: VersionSet{
			{Kind: "ZookeeperCluster", Uid: "001", Version: "500"}: {}, // scale=2
			{Kind: "StatefulSet", Uid: "002", Version: "510"}:      {}, // scale=2
		},
	},
	// NOW THE CONTROLLER CRASHES, GETS RECONNECTED TO STALE APISERVER 2
	{
		// now the controller reads the ZookeeperCluster object from the stale APIServer 2
		// and believes that the desired scale is 1
		// but, it gets StatefulSet from APIServer 1, which has the correct current scale=2
		// so it issues a scale down to the StatefulSet
		timestamp: "t7", elements: VersionSet{
			{Kind: "ZookeeperCluster", Uid: "001", Version: "210"}: {}, // scale=1 (stale)
			{Kind: "StatefulSet", Uid: "002", Version: "510"}:      {}, // scale=2 (current)
		},
	},
	// UPDATE statefulset 002, scale=1
	{
		// observes effect of the action taken at t7
		timestamp: "t8", elements: VersionSet{
			{Kind: "ZookeeperCluster", Uid: "001", Version: "210"}: {}, // scale=1 (stale)
			{Kind: "StatefulSet", Uid: "002", Version: "511"}:      {}, // scale=1 (current)
		},
	},
}

// root cause:
// find the observation that violates the non-decreasing version rule
// show a trace of the events following that violating observation

// function that takes a slice of ObjectVersion and returns true if the versions are non-decreasing
func isNonDecreasing(versions []ObjectVersion) bool {
	for i := 1; i < len(versions); i++ {
		if versions[i].Version < versions[i-1].Version {
			return false
		}
	}
	return true
}

// cassandra-operator 398

// expert context:
// -- in pod deletion, operator needs to be able to observe the deletionTimestamp to handle cleanup tasks properly

// rule: {
//		scope: Kind=pod, uid
// 	    Kubelet and Cassandra-Operator should always observe the same version sequences
// }
// rule stated more weakly (broadly): {
//		scope: Kind=pod, uid
//		Cassandra-operators knowledge of Pod version sequences should be at least as "granular" as anyone else's knowledge
// }
// violation: {
//		kind: Pod
//		uid: 002
// 		interval: [t2, t3]
//      kubelet's view: [44, 45]
//      cassandra-operator's view: [44]
// }

// actual fix:
// add a "finalizer" so kubelet cannot delete the pod until the operator has observed the deletionTimestamp

var kubeletKnowledge = LocalKnowledge{
	{
		timestamp: "t1", elements: VersionSet{
			{Kind: "Pod", Uid: "001", Version: "43"}: {},
			{Kind: "Pod", Uid: "002", Version: "44"}: {},
		},
	},
	{
		timestamp: "t2", elements: VersionSet{
			{Kind: "Pod", Uid: "001", Version: "43"}: {},
			{Kind: "Pod", Uid: "002", Version: "45"}: {}, // marked for deletion
		},
	},
	// DELETE POD 2
	{
		timestamp: "t3", elements: VersionSet{
			{Kind: "Pod", Uid: "001", Version: "43"}: {},
		},
	},
}

var operatorKnowledge = LocalKnowledge{
	{
		timestamp: "t1", elements: VersionSet{
			{Kind: "Pod", Uid: "001", Version: "43"}: {},
			{Kind: "Pod", Uid: "002", Version: "44"}: {},
			{Kind: "PVC", Uid: "011", Version: "56"}: {},
			{Kind: "PVC", Uid: "022", Version: "57"}: {},
		},
	},
	{
		// between t1 and t5, Pod 002 is deleted from etcd, deleted from the operator's cache
		timestamp: "t5", elements: VersionSet{
			{Kind: "Pod", Uid: "001", Version: "43"}: {},
			{Kind: "PVC", Uid: "011", Version: "56"}: {},
			{Kind: "PVC", Uid: "022", Version: "57"}: {}, // orphaned PVC
		},
	},
}

// RobinHood calico outage
//
// expert context:
// -- Pods are tightly coupled to AWS RouteConfigs in their creation and deletion
//
// rule: {
// RouteConfig and Pod objects should always be created and deleted in pairs

var kubeletKnowledge2 = LocalKnowledge{
	{
		// prior to pod creation event
		timestamp: "t1", elements: VersionSet{
			{Kind: "Pod", Uid: "P1", Version: "03", TraceID: "000"}:          {},
			{Kind: "RouteConfig", Uid: "RC1", Version: "07", TraceID: "000"}: {},
		},
	},
	{
		// sees new pod, starts it up, creates a RouteConfig for it
		timestamp: "t2", elements: VersionSet{
			{Kind: "Pod", Uid: "P1", Version: "03", TraceID: "000"}:          {},
			{Kind: "RouteConfig", Uid: "PC1", Version: "07", TraceID: "000"}: {},
			{Kind: "Pod", Uid: "P2", Version: "46", TraceID: "123"}:          {},
		},
	},
	// CREATE RouteConfig-012, UPDATE action
	{
		timestamp: "t2", elements: VersionSet{
			{Kind: "Pod", Uid: "P1", Version: "03", TraceID: "000"}:          {},
			{Kind: "Pod", Uid: "P2", Version: "46", TraceID: "123"}:          {},
			{Kind: "RouteConfig", Uid: "RC1", Version: "07", TraceID: "000"}: {},
			{Kind: "RouteConfig", Uid: "RC2", Version: "47", TraceID: "123"}: {},
		},
	},
}

// expert context:
// -- AWS RouteConfigs are created for a Pod when a pod is created, cleaned up when a Pod is deleted
// -- by the CNI plugin

//	rule: {
//		// RouteConfig should not be created with respect to a Pod
//		// RouteConfig should not be deleted with respect to a Pod
//	}
var calicoKnowledge = LocalKnowledge{
	{
		// prior to pod creation event
		timestamp: "t1", elements: VersionSet{
			{Kind: "Pod", Uid: "P1", Version: "03", TraceID: "000"}:          {},
			{Kind: "RouteConfig", Uid: "RC1", Version: "07", TraceID: "000"}: {},
		},
	},
	{
		// sees new routeConfig before seeing new pod
		timestamp: "t2", elements: VersionSet{
			{Kind: "Pod", Uid: "P1", Version: "03", TraceID: "000"}:          {},
			{Kind: "RouteConfig", Uid: "RC1", Version: "07", TraceID: "000"}: {},
			{Kind: "RouteConfig", Uid: "RC2", Version: "47", TraceID: "123"}: {},
		},
	},
	// decides to delete RouteConfig RC2 b/c it doesn't appear to be associated with a pod
	// DELETE RouteConfig-RC2, DELETE action
	{
		timestamp: "t3", elements: VersionSet{
			{Kind: "Pod", Uid: "P1", Version: "03", TraceID: "000"}:          {},
			{Kind: "RouteConfig", Uid: "RC1", Version: "07", TraceID: "000"}: {},
		},
	},
}
