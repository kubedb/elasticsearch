/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "kubedb.dev/apimachinery/client/clientset/versioned/typed/ops/v1alpha1"

	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeOpsV1alpha1 struct {
	*testing.Fake
}

func (c *FakeOpsV1alpha1) ElasticsearchOpsRequests(namespace string) v1alpha1.ElasticsearchOpsRequestInterface {
	return &FakeElasticsearchOpsRequests{c, namespace}
}

func (c *FakeOpsV1alpha1) EtcdOpsRequests(namespace string) v1alpha1.EtcdOpsRequestInterface {
	return &FakeEtcdOpsRequests{c, namespace}
}

func (c *FakeOpsV1alpha1) MemcachedOpsRequests(namespace string) v1alpha1.MemcachedOpsRequestInterface {
	return &FakeMemcachedOpsRequests{c, namespace}
}

func (c *FakeOpsV1alpha1) MongoDBOpsRequests(namespace string) v1alpha1.MongoDBOpsRequestInterface {
	return &FakeMongoDBOpsRequests{c, namespace}
}

func (c *FakeOpsV1alpha1) MySQLOpsRequests(namespace string) v1alpha1.MySQLOpsRequestInterface {
	return &FakeMySQLOpsRequests{c, namespace}
}

func (c *FakeOpsV1alpha1) PerconaXtraDBOpsRequests(namespace string) v1alpha1.PerconaXtraDBOpsRequestInterface {
	return &FakePerconaXtraDBOpsRequests{c, namespace}
}

func (c *FakeOpsV1alpha1) PgBouncerOpsRequests(namespace string) v1alpha1.PgBouncerOpsRequestInterface {
	return &FakePgBouncerOpsRequests{c, namespace}
}

func (c *FakeOpsV1alpha1) PostgresOpsRequests(namespace string) v1alpha1.PostgresOpsRequestInterface {
	return &FakePostgresOpsRequests{c, namespace}
}

func (c *FakeOpsV1alpha1) ProxySQLOpsRequests(namespace string) v1alpha1.ProxySQLOpsRequestInterface {
	return &FakeProxySQLOpsRequests{c, namespace}
}

func (c *FakeOpsV1alpha1) RedisOpsRequests(namespace string) v1alpha1.RedisOpsRequestInterface {
	return &FakeRedisOpsRequests{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeOpsV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
