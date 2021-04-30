package test_server

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"testing"

	"github.com/confluentinc/kafka-rest-sdk-go/kafkarestv3"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

// Handler for: "/kafka/v3/clusters"
func (r KafkaRestProxyRouter) HandleKafkaRPClusters(t *testing.T) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// List Clusters
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			// Response jsons are created by editting examples from the Kafka REST Proxy OpenApi Spec
			response := `{
				"kind": "KafkaClusterList",
				"metadata": { "self": "http://localhost:8082/v3/clusters", "next": null},
				"data": [
				  {
					"kind": "KafkaCluster","metadata": {"self": "http://localhost:8082/v3/clusters/cluster-1","resource_name": "crn:///kafka=cluster-1"},
					"cluster_id": "cluster-1",
					"controller": {"related": "http://localhost:8082/v3/clusters/cluster-1/brokers/1"},
					"acls": {"related": "http://localhost:8082/v3/clusters/cluster-1/acls"},
					"brokers": {"related": "http://localhost:8082/v3/clusters/cluster-1/brokers"},
					"broker_configs": {"related": "http://localhost:8082/v3/clusters/cluster-1/broker-configs"},
					"consumer_groups": {"related": "http://localhost:8082/v3/clusters/cluster-1/consumer-groups"},
					"topics": {"related": "http://localhost:8082/v3/clusters/cluster-1/topics"},
					"partition_reassignments": {"related": "http://localhost:8082/v3/clusters/cluster-1/topics/-/partitions/-/reassignment"}
				  }
				]
			}`
			_, err := io.WriteString(w, response)
			require.NoError(t, err)
		}
	}
}

// Handler for: "/kafka/v3/clusters/{cluster}/acls"
func (r KafkaRestProxyRouter) HandleKafkaRPACLs(t *testing.T) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.Header().Set("Content-Type", "application/json")
			vars := mux.Vars(r)
			err := json.NewEncoder(w).Encode(kafkarestv3.AclDataList{Data: []kafkarestv3.AclData{{
				Kind:         "",
				Metadata:     kafkarestv3.ResourceMetadata{},
				ClusterId:    vars["cluster"],
				ResourceType: "TOPIC",
				ResourceName: "test-rest-proxy-topic",
				Operation:    "READ",
				Permission:   "ALLOW",
			}}})
			require.NoError(t, err)
		case "POST":
			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json")
			var req kafkarestv3.ClustersClusterIdAclsPostOpts
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)
			err = json.NewEncoder(w).Encode(kafkarestv3.ClustersClusterIdAclsPostOpts{})
			require.NoError(t, err)
		case "DELETE":
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(kafkarestv3.InlineResponse200{Data: []kafkarestv3.AclData{{ResourceName: "topic-1"}}})
			require.NoError(t, err)
		}
	}
}

// Handler for: "/kafka/v3/clusters/{cluster}/topics"
func (r KafkaRestProxyRouter) HandleKafkaRPTopics(t *testing.T) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.Header().Set("Content-Type", "application/json")
			response := `{
				"kind": "KafkaTopicList",
				"metadata": {"self": "http://localhost:9391/v3/clusters/cluster-1/topics","next": null},
				"data": [
				  {
					"kind": "KafkaTopic",
					"metadata": {"self": "http://localhost:9391/v3/clusters/cluster-1/topics/topic-1","resource_name": "crn:///kafka=cluster-1/topic=topic-1"},
					"cluster_id": "cluster-1",
					"topic_name": "topic-1",
					"is_internal": false,
					"replication_factor": 3,
					"partitions": {"related": "http://localhost:9391/v3/clusters/cluster-1/topics/topic-1/partitions"},
					"configs": {"related": "http://localhost:9391/v3/clusters/cluster-1/topics/topic-1/configs"},
					"partition_reassignments": {"related": "http://localhost:9391/v3/clusters/cluster-1/topics/topic-1/partitions/-/reassignments"}
				  },
				  {
					"kind": "KafkaTopic",
					"metadata": {"self": "http://localhost:9391/v3/clusters/cluster-1/topics/topic-2","resource_name": "crn:///kafka=cluster-1/topic=topic-2"},
					"cluster_id": "cluster-1",
					"topic_name": "topic-2",
					"is_internal": true,
					"replication_factor": 4,
					"partitions": {"related": "http://localhost:9391/v3/clusters/cluster-1/topics/topic-2/partitions"},
					"configs": {"related": "http://localhost:9391/v3/clusters/cluster-1/topics/topic-2/configs"},
					"partition_reassignments": {"related": "http://localhost:9391/v3/clusters/cluster-1/topics/topic-2/partitions/-/reassignments"}
				  }
				]
			}`
			_, err := io.WriteString(w, response)
			require.NoError(t, err)
		case "POST":
			// Parse Create Args
			reqBody, _ := ioutil.ReadAll(r.Body)
			var requestData kafkarestv3.CreateTopicRequestData
			err := json.Unmarshal(reqBody, &requestData)
			require.NoError(t, err)
			if requestData.TopicName == "topic-exist" { // check topic
				require.NoError(t, writeErrorResponse(w, http.StatusBadRequest, 40002, "Topic 'topic-exist' already exists."))
				return
			} else if requestData.PartitionsCount < -1 || requestData.PartitionsCount == 0 { // check partition
				require.NoError(t, writeErrorResponse(w, http.StatusBadRequest, 40002, "Number of partitions must be larger than 0."))
				return
			} else if requestData.ReplicationFactor < -1 || requestData.ReplicationFactor == 0 { // check replication factor
				require.NoError(t, writeErrorResponse(w, http.StatusBadRequest, 40002, "Replication factor must be larger than 0."))
				return
			} else if requestData.ReplicationFactor > 3 {
				require.NoError(t, writeErrorResponse(w, http.StatusBadRequest, 40002, "Replication factor: 4 larger than available brokers: 3."))
				return
			}
			// check configs
			for _, config := range requestData.Configs {
				if config.Name != "retention.ms" && config.Name != "compression.type" {
					require.NoError(t, writeErrorResponse(w, http.StatusBadRequest, 40002, fmt.Sprintf("Unknown topic config name: %s", config.Name)))
					return
				} else if config.Name == "retention.ms" {
					if config.Value == nil { // if retention.ms but value null
						require.NoError(t, writeErrorResponse(w, http.StatusBadRequest, 40002, "Null value not supported for topic configs : retention.ms"))
						return
					} else if _, err := strconv.Atoi(*config.Value); err != nil { // if retention.ms but value invalid
						require.NoError(t, writeErrorResponse(w, http.StatusBadRequest, 40002, fmt.Sprintf("Invalid value %s for configuration retention.ms: Not a number of type LONG", *config.Value)))
						return
					}
				}
				// TODO: check for compression.type
			}
			// no errors = successfully created
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			response := fmt.Sprintf(`{
					"kind": "KafkaTopic",
					"metadata": {
					  "self": "http://localhost:9391/v3/clusters/cluster-1/topics/%[1]s",
					  "resource_name": "crn:///kafka=cluster-1/topic=%[1]s"
					},
					"cluster_id": "cluster-1",
					"topic_name": "%[1]s",
					"is_internal": false,
					"replication_factor": %[2]d,
					"partitions": {
					  "related": "http://localhost:9391/v3/clusters/cluster-1/topics/%[1]s/partitions"
					},
					"configs": {
					  "related": "http://localhost:9391/v3/clusters/cluster-1/topics/%[1]s/configs"
					},
					"partition_reassignments": {
					  "related": "http://localhost:9391/v3/clusters/cluster-1/topics/%[1]s/partitions/-/reassignments"
					}
				  }`, requestData.TopicName, requestData.ReplicationFactor)
			_, err = io.WriteString(w, response)
			require.NoError(t, err)
		}
	}
}

// Handler for: "/kafka/v3/clusters/{cluster}/topics/{topic}/partitions"
func (r KafkaRestProxyRouter) HandleKafkaRPPartitions(t *testing.T) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		topicName := vars["topic"]
		switch r.Method {
		case "GET":
			if topicName == "topic-exist" {
				w.Header().Set("Content-Type", "application/json")
				responseString := fmt.Sprintf(`{
					"kind": "KafkaPartitionList",
					"metadata": {
						"self": "http://localhost:8082/v3/clusters/cluster-1/topics/%[1]s/partitions",
						"next": null
					},
					"data": [
						{
							"kind": "KafkaPartition",
							"metadata": {
								"self": "http://localhost:8082/v3/clusters/cluster-1/topics/%[1]s/partitions/0",
								"resource_name": "crn:///kafka=cluster-1/topic=%[1]s/partition=0"
							},
							"cluster_id": "cluster-1",
							"topic_name": "%[1]s",
							"partition_id": 0,
							"leader": {"related": "http://localhost:8082/v3/clusters/cluster-1/topics/%[1]s/partitions/0/replicas/1001"},
							"replicas": {"related": "http://localhost:8082/v3/clusters/cluster-1/topics/%[1]s/partitions/0/replicas"},
							"reassignment": {"related": "http://localhost:8082/v3/clusters/cluster-1/topics/%[1]s/partitions/0/reassignment"}
						},
						{
							"kind": "KafkaPartition",
							"metadata": {
								"self": "http://localhost:8082/v3/clusters/cluster-1/topics/%[1]s/partitions/1",
								"resource_name": "crn:///kafka=cluster-1/topic=%[1]s/partition=1"
							},
							"cluster_id": "cluster-1",
							"topic_name": "%[1]s",
							"partition_id": 1,
							"leader": {"related": "http://localhost:8082/v3/clusters/cluster-1/topics/%[1]s/partitions/1/replicas/1001"},
							"replicas": {"related": "http://localhost:8082/v3/clusters/cluster-1/topics/%[1]s/partitions/1/replicas"},
							"reassignment": {"related": "http://localhost:8082/v3/clusters/cluster-1/topics/%[1]s/partitions/1/reassignment"}
						},
						{
							"kind": "KafkaPartition",
							"metadata": {
								"self": "http://localhost:8082/v3/clusters/cluster-1/topics/%[1]s/partitions/2",
								"resource_name": "crn:///kafka=cluster-1/topic=%[1]s/partition=2"
							},
							"cluster_id": "cluster-1",
							"topic_name": "%[1]s",
							"partition_id": 2,
							"leader": {"related": "http://localhost:8082/v3/clusters/cluster-1/topics/%[1]s/partitions/2/replicas/1001"},
							"replicas": {"related": "http://localhost:8082/v3/clusters/cluster-1/topics/%[1]s/partitions/2/replicas"},
							"reassignment": {"related": "http://localhost:8082/v3/clusters/cluster-1/topics/%[1]s/partitions/2/reassignment"}
						}
					]
				}`, "topic-exist")
				_, err := io.WriteString(w, responseString)
				require.NoError(t, err)
			} else {
				require.NoError(t, writeErrorResponse(w, http.StatusNotFound, 40403, "This server does not host this topic-partition."))
			}
		}
	}
}

// Handler for: "/kafka/v3/clusters/{cluster}/topics/{topic}/configs"
func (r KafkaRestProxyRouter) HandleKafkaRPTopicConfigs(t *testing.T) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		topicName := vars["topic"]
		switch r.Method {
		case "GET":
			// if topic exists
			if topicName == "topic-exist" {
				responseString := fmt.Sprintf(`{
					"kind": "KafkaTopicConfigList",
					"metadata": {
						"self": "http://localhost:8082/v3/clusters/cluster-1/topics/topic-exist/configs",
						"next": null
					},
					"data": [
						{
							"kind": "KafkaTopicConfig",
							"metadata": {
								"self": "http://localhost:8082/v3/clusters/cluster-1/topics/topic-exist/configs/cleanup.policy",
								"resource_name": "crn:///kafka=cluster-1/topic=topic-exist/config=cleanup.policy"
							},
							"cluster_id": "cluster-1",
							"name": "cleanup.policy",
							"value": "delete",
							"is_read_only": false,
							"is_sensitive": false,
							"source": "DEFAULT_CONFIG",
							"synonyms": [
								{
									"name": "log.cleanup.policy",
									"value": "delete",
									"source": "DEFAULT_CONFIG"
								}
							],
							"topic_name": "topic-exist",
							"is_default": true
						},
						{
							"kind": "KafkaTopicConfig",
							"metadata": {
								"self": "http://localhost:8082/v3/clusters/cluster-1/topics/topic-exist/configs/compression.type",
								"resource_name": "crn:///kafka=cluster-1/topic=topic-exist/config=compression.type"
							},
							"cluster_id": "cluster-1",
							"name": "compression.type",
							"value": "producer",
							"is_read_only": false,
							"is_sensitive": false,
							"source": "DEFAULT_CONFIG",
							"synonyms": [
								{
									"name": "compression.type",
									"value": "producer",
									"source": "DEFAULT_CONFIG"
								}
							],
							"topic_name": "topic-exist",
							"is_default": true
						},
						{
							"kind": "KafkaTopicConfig",
							"metadata": {
								"self": "http://localhost:8082/v3/clusters/cluster-1/topics/topic-exist/configs/retention.ms",
								"resource_name": "crn:///kafka=cluster-1/topic=topic-exist/config=retention.ms"
							},
							"cluster_id": "cluster-1",
							"name": "retention.ms",
							"value": "604800000",
							"is_read_only": false,
							"is_sensitive": false,
							"source": "DEFAULT_CONFIG",
							"synonyms": [],
							"topic_name": "topic-exist",
							"is_default": true
						}
					]
				}`)

				w.Header().Set("Content-Type", "application/json")
				_, err := io.WriteString(w, responseString)
				require.NoError(t, err)

			} else { // if topic not exist
				require.NoError(t, writeErrorResponse(w, http.StatusNotFound, 40403, "This server does not host this topic-partition."))
			}
		}
	}
}

// Handler for: "/kafka/v3/clusters/{cluster}/topics/{topic}/partitions/{partition}/replicas"
func (r KafkaRestProxyRouter) HandleKafkaRPPartitionReplicas(t *testing.T) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		topicName := vars["topic"]
		partitionId := vars["partition"]
		switch r.Method {
		case "GET":
			// if topic exists
			if topicName == "topic-exist" {
				// Define replica & partition info
				type replicaData struct {
					brokerId int
					isLeader bool
					isInSync bool
				}
				partitionInfo := map[string]struct { // TODO: add test for different # of replicas for different partitions
					replicas []replicaData
				}{
					"0": {
						replicas: []replicaData{{brokerId: 1001, isLeader: true, isInSync: true},
							{brokerId: 1002, isLeader: false, isInSync: true},
							{brokerId: 1003, isLeader: false, isInSync: true}},
					},
					"1": {
						replicas: []replicaData{{brokerId: 1001, isLeader: false, isInSync: false},
							{brokerId: 1002, isLeader: true, isInSync: true},
							{brokerId: 1003, isLeader: false, isInSync: true}},
					},
					"2": {
						replicas: []replicaData{{brokerId: 1001, isLeader: false, isInSync: false},
							{brokerId: 1002, isLeader: false, isInSync: false},
							{brokerId: 1003, isLeader: true, isInSync: true}},
					},
				}

				// Build response string
				// Different sets of replica strings for different partitions
				replicaString := make([]string, len(partitionInfo[partitionId].replicas))
				for i := range partitionInfo[partitionId].replicas {
					replicaString[i] = fmt.Sprintf(`{
						"kind": "KafkaReplica",
						"metadata": {
							"self": "http://localhost:8082/v3/clusters/cluster-1/topics/%[1]s/partitions/%[2]s/replicas/1001",
							"resource_name": "crn:///kafka=cluster-1/topic=%[1]s/partition=%[2]s/replica=1001"
						},
						"cluster_id": "cluster-1",
						"topic_name": "%[1]s",
						"partition_id": %[2]s,
						"broker_id": %[3]d,
						"is_leader": %[4]t,
						"is_in_sync": %[5]t,
						"broker": {
							"related": "http://localhost:8082/v3/clusters/cluster-1/brokers/1001"
						}
					}`, "topic-exist", partitionId, partitionInfo[partitionId].replicas[i].brokerId,
						partitionInfo[partitionId].replicas[i].isLeader,
						partitionInfo[partitionId].replicas[i].isInSync)
				}
				responseString := fmt.Sprintf(`{
					"kind": "KafkaReplicaList",
					"metadata": {
						"self": "http://localhost:8082/v3/clusters/cluster-1/topics/%[1]s/partitions/%[2]s/replicas",
						"next": null
					},
					"data": [
						%[3]s,
						%[4]s,
						%[5]s
					]
				}`, "topic-exist", partitionId, replicaString[0], replicaString[1], replicaString[2])

				w.Header().Set("Content-Type", "application/json")
				_, err := io.WriteString(w, responseString)
				require.NoError(t, err)
			} else { // if topic not exist
				require.NoError(t, writeErrorResponse(w, http.StatusNotFound, 40403, "This server does not host this topic-partition."))
			}
		}
	}
}

// Handler for: "/kafka/v3/clusters/{cluster}/topics/{topic}/configs:alter"
func (r KafkaRestProxyRouter) HandleKafkaRPConfigsAlter(t *testing.T) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		topicName := vars["topic"]
		switch r.Method {
		case "POST":
			if topicName == "topic-exist" {
				// Parse Alter Args
				requestBody, err := ioutil.ReadAll(r.Body)
				require.NoError(t, err)
				var requestData kafkarestv3.AlterConfigBatchRequestData
				err = json.Unmarshal(requestBody, &requestData)
				require.NoError(t, err)

				// Check Alter Args if valid
				for _, config := range requestData.Data {
					if config.Name != "retention.ms" && config.Name != "compression.type" { // should be either retention.ms or compression.type
						require.NoError(t, writeErrorResponse(w, http.StatusNotFound, 404, fmt.Sprintf("Config %s cannot be found for TOPIC topic-exist in cluster cluster-1.", config.Name)))
						return
					} else if config.Name == "retention.ms" {
						if config.Value == nil { // if retention.ms but value null
							require.NoError(t, writeErrorResponse(w, http.StatusBadRequest, 40002, "Null value not supported for : SET:retention.ms"))
							return
						} else if _, err := strconv.Atoi(*config.Value); err != nil { // if retention.ms but value invalid
							require.NoError(t, writeErrorResponse(w, http.StatusBadRequest, 40002, fmt.Sprintf("Invalid config value for resource ConfigResource(type=TOPIC, name='topic-exist'): Invalid value %s for configuration retention.ms: Not a number of type LONG", *config.Value)))
							return
						}
					}
					// TODO check for compression.type values
				}
				// No error
				w.WriteHeader(http.StatusNoContent)
			} else { // topic-not-exist
				// not found
				require.NoError(t, writeErrorResponse(w, http.StatusNotFound, 40403, "This server does not host this topic-partition."))
			}
		}
	}
}

// Handler for: "/kafka/v3/clusters/{cluster}/topics/{topic}"
func (r KafkaRestProxyRouter) HandleKafkaRPTopic(t *testing.T) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		switch r.Method {
		case "DELETE":
			if vars["topic"] == "topic-exist" {
				// Successfully deleted
				w.WriteHeader(http.StatusNoContent)
				w.Header().Set("Content-Type", "application/json")
			} else {
				// topic not found
				require.NoError(t, writeErrorResponse(w, http.StatusNotFound, 40403, "This server does not host this topic-partition."))
			}

		}
	}
}

// Handler for: "/kafka/v3/clusters/{cluster_id}/consumer-groups"
func (r KafkaRestProxyRouter) HandleKafkaRPConsumerGroups(t *testing.T) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(kafkarestv3.ConsumerGroupDataList{
				Kind:     "",
				Metadata: kafkarestv3.ResourceCollectionMetadata{},
				Data: []kafkarestv3.ConsumerGroupData{
					{
						Kind:              "",
						Metadata:          kafkarestv3.ResourceMetadata{},
						ClusterId:         "cluster-1",
						ConsumerGroupId:   "consumer-group-1",
						IsSimple:          true,
						PartitionAssignor: "org.apache.kafka.clients.consumer.RoundRobinAssignor",
						State:             kafkarestv3.CONSUMERGROUPSTATE_STABLE,
						Coordinator:       kafkarestv3.Relationship{},
						Consumer:          kafkarestv3.Relationship{},
						LagSummary:        kafkarestv3.Relationship{},
					},
					{
						Kind:              "",
						Metadata:          kafkarestv3.ResourceMetadata{},
						ClusterId:         "cluster-1",
						ConsumerGroupId:   "consumer-group-2",
						IsSimple:          true,
						PartitionAssignor: "org.apache.kafka.clients.consumer.RoundRobinAssignor",
						State:             kafkarestv3.CONSUMERGROUPSTATE_DEAD,
						Coordinator:       kafkarestv3.Relationship{},
						Consumer:          kafkarestv3.Relationship{},
						LagSummary:        kafkarestv3.Relationship{},
					},
				},
			})
			require.NoError(t, err)
		}
	}
}

// Handler for: "/kafka/v3/clusters/{cluster_id}/consumer-groups/{consumer_group_id}"
func (r KafkaRestProxyRouter) HandleKafkaRPConsumerGroup(t *testing.T) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		switch r.Method {
		case "GET":
			if vars["consumer_group_id"] == "consumer-group-1" {
				w.Header().Set("Content-Type", "application/json")
				err := json.NewEncoder(w).Encode(kafkarestv3.ConsumerGroupData{
					Kind:              "",
					Metadata:          kafkarestv3.ResourceMetadata{},
					ClusterId:         "cluster-1",
					ConsumerGroupId:   "consumer-group-1",
					IsSimple:          true,
					PartitionAssignor: "RoundRobin",
					State:             kafkarestv3.CONSUMERGROUPSTATE_STABLE,
					Coordinator:       kafkarestv3.Relationship{Related: "/kafka/v3/clusters/cluster-1/brokers/broker-1"},
					Consumer:          kafkarestv3.Relationship{},
					LagSummary:        kafkarestv3.Relationship{},
				})
				require.NoError(t, err)
			} else {
				// group not found
				require.NoError(t, writeErrorResponse(w, http.StatusNotFound, 40403, "This server does not host this consumer group."))
			}
		}
	}
}

// Handler for: "/kafka/v3/clusters/{cluster_id}/consumer-groups/{consumer_group_id}/consumers"
func (r KafkaRestProxyRouter) HandleKafkaRPConsumers(t *testing.T) func(http.ResponseWriter, *http.Request) {
	instance1 := "instance-1"
	instance2 := "instance-2"
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(kafkarestv3.ConsumerDataList{
				Kind:     "",
				Metadata: kafkarestv3.ResourceCollectionMetadata{},
				Data: []kafkarestv3.ConsumerData{
					{
						Kind:            "",
						Metadata:        kafkarestv3.ResourceMetadata{},
						ClusterId:       "cluster-1",
						ConsumerGroupId: "consumer-group-1",
						ConsumerId:      "consumer-1",
						InstanceId:      &instance1,
						ClientId:        "client-1",
						Assignments:     kafkarestv3.Relationship{},
					},
					{
						Kind:            "",
						Metadata:        kafkarestv3.ResourceMetadata{},
						ClusterId:       "cluster-1",
						ConsumerGroupId: "consumer-group-1",
						ConsumerId:      "consumer-2",
						InstanceId:      &instance2,
						ClientId:        "client-2",
						Assignments:     kafkarestv3.Relationship{},
					},
				},
			})
			require.NoError(t, err)
		}
	}
}

// Handler for: "/kafka/v3/clusters/{cluster_id}/consumer-groups/{consumer_group_id}/lag-summary"
func (r KafkaRestProxyRouter) HandleKafkaRPLagSummary(t *testing.T) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		switch r.Method {
		case "GET":
			if vars["consumer_group_id"] == "consumer-group-1" {
				w.Header().Set("Content-Type", "application/json")
				instance := "instance-1"
				err := json.NewEncoder(w).Encode(kafkarestv3.ConsumerGroupLagSummaryData{
					Kind:              "",
					Metadata:          kafkarestv3.ResourceMetadata{},
					ClusterId:         "cluster-1",
					ConsumerGroupId:   "consumer-group-1",
					MaxLagConsumerId:  "consumer-1",
					MaxLagInstanceId:  &instance,
					MaxLagClientId:    "client-1",
					MaxLagTopicName:   "topic-1",
					MaxLagPartitionId: 1,
					MaxLag:            100,
					TotalLag:          110,
					MaxLagConsumer:    kafkarestv3.Relationship{},
					MaxLagPartition:   kafkarestv3.Relationship{},
				})
				require.NoError(t, err)
			} else {
				// group not found
				require.NoError(t, writeErrorResponse(w, http.StatusNotFound, 40403, "This server does not host this consumer group."))
			}
		}
	}
}

// Handler for: "/kafka/v3/clusters/{cluster_id}/consumer-groups/{consumer_group_id}/lags"
func (r KafkaRestProxyRouter) HandleKafkaRPLags(t *testing.T) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		switch r.Method {
		case "GET":
			if vars["consumer_group_id"] == "consumer-group-1" {
				w.Header().Set("Content-Type", "application/json")
				instance1 := "instance-1"
				instance2 := "instance-2"
				err := json.NewEncoder(w).Encode(kafkarestv3.ConsumerLagDataList{
					Kind:     "",
					Metadata: kafkarestv3.ResourceCollectionMetadata{},
					Data: []kafkarestv3.ConsumerLagData{
						{
							Kind:            "",
							Metadata:        kafkarestv3.ResourceMetadata{},
							ClusterId:       "cluster-1",
							ConsumerGroupId: "consumer-group-1",
							TopicName:       "topic-1",
							PartitionId:     1,
							CurrentOffset:   1,
							LogEndOffset:    101,
							Lag:             100,
							ConsumerId:      "consumer-1",
							InstanceId:      &instance1,
							ClientId:        "client-1",
						},
						{
							Kind:            "",
							Metadata:        kafkarestv3.ResourceMetadata{},
							ClusterId:       "cluster-1",
							ConsumerGroupId: "consumer-group-1",
							TopicName:       "topic-1",
							PartitionId:     2,
							CurrentOffset:   1,
							LogEndOffset:    11,
							Lag:             10,
							ConsumerId:      "consumer-2",
							InstanceId:      &instance2,
							ClientId:        "client-2",
						},
					},
				})
				require.NoError(t, err)
			} else {
				// group not found
				require.NoError(t, writeErrorResponse(w, http.StatusNotFound, 40403, "This server does not host this consumer group."))
			}
		}
	}
}

type partitionOffsets struct {
	currentOffset int32
	logEndOffset  int32
}

// Handler for: "/kafka/v3/clusters/{cluster_id}/consumer-groups/{consumer_group_id}/lags/{topic_name}/partitions/{partition_id}"
func (r KafkaRestProxyRouter) HandleKafkaRPLag(t *testing.T) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		fmt.Println(vars)
		switch r.Method {
		case "GET":
			if vars["consumer_group_id"] == "consumer-group-1" {
				partitionOffsetsMap := map[string]partitionOffsets{
					"0": {101, 101},
					"1": {1, 101},
					"2": {101, 101},
				}
				requestedPartition := vars["partition_id"]
				offsets := partitionOffsetsMap[requestedPartition]
				if vars["topic_name"] == "topic-1" && offsets != (partitionOffsets{}) {
					w.Header().Set("Content-Type", "application/json")
					instance := "instance-1"
					partitionId, _ := strconv.Atoi(requestedPartition)
					err := json.NewEncoder(w).Encode(kafkarestv3.ConsumerLagData{
						Kind:            "",
						Metadata:        kafkarestv3.ResourceMetadata{},
						ClusterId:       "cluster-1",
						ConsumerGroupId: "consumer-group-1",
						TopicName:       "topic-1",
						PartitionId:     int32(partitionId),
						CurrentOffset:   offsets.currentOffset,
						LogEndOffset:    offsets.logEndOffset,
						Lag:             offsets.logEndOffset - offsets.currentOffset,
						ConsumerId:      "consumer-1",
						InstanceId:      &instance,
						ClientId:        "client-1",
					})
					require.NoError(t, err)
				} else {
					// topic and/or partition not found
					require.NoError(t, writeErrorResponse(w, http.StatusNotFound, 40403, "This server does not host this topic-partition."))
				}
			} else {
				// group not found
				require.NoError(t, writeErrorResponse(w, http.StatusNotFound, 40403, "This server does not host this consumer group."))
			}
		}
	}
}

func writeErrorResponse(responseWriter http.ResponseWriter, statusCode int, errorCode int, message string) error {
	responseWriter.WriteHeader(statusCode)
	responseBody := fmt.Sprintf(`{
		"error_code": %d,
		"message": "%s"
	}`, errorCode, message)
	_, err := io.WriteString(responseWriter, responseBody)
	return err
}
