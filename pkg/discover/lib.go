package discover

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/appscode/go/log"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func write(path, data string) {
	ensureDirectory(path)
	if err := ioutil.WriteFile(path, []byte(data), os.ModePerm); err != nil {
		log.Fatal(err)
	}
	return
}

func ensureDirectory(path string) {
	parent := filepath.Dir(path)
	if _, err := os.Stat(parent); err != nil {
		if err = os.MkdirAll(parent, os.ModePerm); err != nil {
			log.Fatal(err)
		}
	}
}

func flattenSubsets(subsets []apiv1.EndpointSubset) []string {
	ips := []string{}
	for _, ss := range subsets {
		for _, addr := range ss.Addresses {
			ips = append(ips, addr.IP)
		}
	}
	return ips
}

func DiscoverEndpoints(config *rest.Config, service, namespace string) {
	log.Info("Kubernetes Elasticsearch Cluster discovery")
	log.Infof("Searching for %s.%s", service, namespace)
	////////////////////////////////////////////////

	c, err := clientset.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to make client: %v", err)
	}

	var elasticsearch *apiv1.Service
	// Look for endpoints associated with the Elasticsearch loggging service.
	// First wait for the service to become available.
	for t := time.Now(); time.Since(t) < 5*time.Minute; time.Sleep(10 * time.Second) {
		elasticsearch, err = c.CoreV1().Services(namespace).Get(service, metav1.GetOptions{})
		if err == nil {
			break
		}
	}
	// If we did not find an elasticsearch logging service then log a warning
	// and return without adding any unicast hosts.
	if elasticsearch == nil {
		log.Warningf("Failed to find the Kubernetes service: %v", err)
		return
	}

	var endpoints *apiv1.Endpoints
	addrs := []string{}

	// $(statefulset name)-$(ordinal)
	podName := os.Getenv("POD_NAME")
	ignoreWaiting := false
	if podName != "" {
		parts := strings.Split(podName, "-")
		if len(parts) != 0 {
			if ordinal, err := strconv.Atoi(parts[len(parts)-1]); err == nil && ordinal == 0 {
				// Count it as a first node of Elasticsearch cluster.
				ignoreWaiting = true
			}
		}
	}

	// Wait for some endpoints.
	count := 0
	for t := time.Now(); time.Since(t) < 5*time.Minute; time.Sleep(10 * time.Second) {
		endpoints, err = c.CoreV1().Endpoints(namespace).Get(service, metav1.GetOptions{})
		if err != nil {
			continue
		}

		addrs = flattenSubsets(endpoints.Subsets)
		log.Infof("Found %s", addrs)
		if len(addrs) > 0 && len(addrs) == count {
			break
		}
		count = len(addrs)

		if ignoreWaiting {
			break
		}
	}
	// If there was an error finding endpoints then log a warning and quit.
	if err != nil {
		log.Warningf("Error finding endpoints: %v", err)
		return
	}

	endpointsDNS := make([]string, 0)
	if len(addrs) > 0 {
		for _, ip := range addrs {
			log.Debugln(fmt.Sprintf(`Lookup address for IP "%v"`, ip))
			dnsName, err := net.LookupAddr(ip)
			if err != nil {
				log.Errorln(err)
				continue
			}
			endpointsDNS = append(endpointsDNS, dnsName...)
		}
		if len(endpointsDNS) == 0 {
			endpointsDNS = addrs
			log.Debugln("dns address not found. Using IPs")
		} else {
			log.Debugln("Found dns address")
		}
		log.Infof("Endpoints = %s", endpointsDNS)
	}

	path := "/tmp/discovery/unicast-hosts"
	data := fmt.Sprintf("discovery.zen.ping.unicast.hosts: [%s]\n", strings.Join(endpointsDNS, ", "))
	write(path, data)
}
