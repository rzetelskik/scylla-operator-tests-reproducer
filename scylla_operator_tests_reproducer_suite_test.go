// Copyright (C) 2023 ScyllaDB

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	x "github.com/onsi/gomega/gexec"
)

func TestScyllaOperatorTestsReproducer(t *testing.T) {
	o.RegisterFailHandler(g.Fail)

	g.RunSpecs(t, "ScyllaOperatorTestsReproducer Suite")
}

var _ = g.Describe("Scylla Operator Tests Reproducer", func() {
	execPath := getEnvOrDefault("SCYLLA_OPERATOR_TESTS_REPRODUCER_EXEC", "./scylla-operator-tests")

	artifactsDir := getEnvOrDefault("SCYLLA_OPERATOR_TESTS_REPRODUCER_ARTIFACTS_DIR", os.TempDir())
	cmd := exec.Command(execPath, "run", "scylla-operator/conformance/parallel", fmt.Sprintf("--artifacts-dir=%s", artifactsDir), `--delete-namespace-policy=Always`, `--dry-run=true`, `--fail-fast=false`, `--flake-attempts=0`, `--loglevel=2`, `--parallelism=0`, `--progress=true`)

	session, err := x.Start(cmd, g.GinkgoWriter, g.GinkgoWriter)
	o.Expect(err).NotTo(o.HaveOccurred())
	o.Eventually(session, 1*time.Minute).Should(x.Exit(0))

	jsonReport := path.Join(artifactsDir, "e2e.json")
	o.Expect(jsonReport).To(o.BeAnExistingFile())
	f, err := os.Open(jsonReport)
	o.Expect(err).NotTo(o.HaveOccurred())

	bytes, err := io.ReadAll(f)
	o.Expect(err).NotTo(o.HaveOccurred())

	discard := make([]g.Report, 0)
	err = json.Unmarshal(bytes, &discard)
	o.Expect(err).NotTo(o.HaveOccurred())
	o.Expect(discard).To(o.HaveLen(1))

	report := discard[0]
	var specsRan []string
	for _, sr := range report.SpecReports {
		specsRan = append(specsRan, sr.FullText())
	}

	expectedSpecs := []string{
		"Node Setup should make RAID0, format it to XFS, and mount at desired location out of one loop device",
		"Node Setup should make RAID0, format it to XFS, and mount at desired location out of three loop devices",
		"NodeConfig Optimizations should correctly project state for each scylla pod",
		"NodeConfig Optimizations should create tuning resources and tune nodes",
		"Scylla Manager integration should discover cluster and sync tasks",
		"ScyllaCluster HostID should be reflected as a Service annotation",
		"ScyllaCluster Ingress should create ingress objects when ingress exposeOptions are provided",
		"ScyllaCluster Orphaned PV controller should replace a node with orphaned PV",
		"ScyllaCluster authentication agent requires authentication",
		"ScyllaCluster evictions should allow one disruption",
		"ScyllaCluster replace should replace a node",
		"ScyllaCluster should allow to build connection pool using shard aware ports",
		"ScyllaCluster should claim preexisting member ServiceAccount and RoleBinding",
		"ScyllaCluster should re-bootstrap from old PVCs",
		"ScyllaCluster should reconcile resource changes",
		"ScyllaCluster should rotate TLS certificates before they expire",
		"ScyllaCluster should setup and maintain up to date TLS certificates",
		"ScyllaCluster should support scaling",
		"ScyllaCluster sysctl should set container sysctl",
		"ScyllaCluster upgrades should deploy and update with 1 member(s) and 1 rack(s) from 5.0.12 to 5.1.9",
		"ScyllaCluster upgrades should deploy and update with 1 member(s) and 1 rack(s) from 5.1.8 to 5.1.9",
		"ScyllaCluster upgrades should deploy and update with 3 member(s) and 1 rack(s) from 5.0.12 to 5.1.9",
		"ScyllaCluster upgrades should deploy and update with 3 member(s) and 1 rack(s) from 5.1.8 to 5.1.9",
		"ScyllaCluster upgrades should deploy and update with 3 member(s) and 2 rack(s) from 5.0.12 to 5.1.9",
		"ScyllaCluster webhook should forbid invalid requests",
		"ScyllaDBMonitoring should setup monitoring stack",
	}

	g.It("Should run all specs", func() {
		o.Expect(specsRan).To(o.ConsistOf(expectedSpecs))
	})

	specsRanMap := make(map[string][]int, 0)
	for _, sr := range []g.SpecReport(report.SpecReports) {
		specsRanMap[sr.FullText()] = append(specsRanMap[sr.FullText()], sr.ParallelProcess)
	}

	entriesFn := func() []g.TableEntry {
		entries := make([]g.TableEntry, 0)
		for _, e := range expectedSpecs {
			k := e
			v := specsRanMap[k]
			entries = append(entries, g.Entry(func(_ []int) string {
				return fmt.Sprintf("%q on a single worker, but ran it on workers %v", k, v)
			}, v))
		}

		return entries
	}

	g.DescribeTable("Should run spec", func(v []int) {
		o.Expect(v).To(o.HaveLen(1))
	}, entriesFn())
})

func getEnvOrDefault(key, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if ok {
		return value
	}

	return defaultValue
}
