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
	"golang.org/x/exp/maps"
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
	leafNodeTextsRanMap := make(map[string][]int, 0)
	for _, sr := range []g.SpecReport(report.SpecReports) {
		leafNodeTextsRanMap[sr.LeafNodeText] = append(leafNodeTextsRanMap[sr.LeafNodeText], sr.ParallelProcess)
	}

	expectedLeafNodeTexts := []string{
		`should allow to build connection pool using shard aware ports`,
		`should create tuning resources and tune nodes`,
		`should replace a node with orphaned PV`,
		`should replace a node`,
		`should setup and maintain up to date TLS certificates`,
		`should forbid invalid requests`,
		`with 3 member(s) and 1 rack(s) from 5.1.8 to 5.1.9`,
		`should be reflected as a Service annotation`,
		`with 1 member(s) and 1 rack(s) from 5.1.8 to 5.1.9`,
		`should set container sysctl`,
		`should re-bootstrap from old PVCs`,
		`should support scaling`,
		`with 3 member(s) and 2 rack(s) from 5.1.9 to 5.2.0`,
		`should rotate TLS certificates before they expire`,
		`with 3 member(s) and 1 rack(s) from 5.1.9 to 5.2.0`,
		`with 1 member(s) and 1 rack(s) from 5.1.9 to 5.2.0`,
		`should create ingress objects when ingress exposeOptions are provided`,
		`should allow one disruption`,
		`should discover cluster and sync tasks`,
		`agent requires authentication`,
		`should setup monitoring stack`,
		`should reconcile resource changes`,
		`should correctly project state for each scylla pod`,
		`should claim preexisting member ServiceAccount and RoleBinding`,
	}

	g.It("Should run all specs", func() {
		o.Expect(maps.Keys(leafNodeTextsRanMap)).To(o.ConsistOf(expectedLeafNodeTexts))
	})

	entriesFn := func() []g.TableEntry {
		entries := make([]g.TableEntry, 0)
		for _, e := range expectedLeafNodeTexts {
			k := e
			v := leafNodeTextsRanMap[k]
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
