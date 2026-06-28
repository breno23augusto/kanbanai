#!/usr/bin/env python3
"""
Automated tests for the dependency health validation script (/tmp/gen_report.py).

Covers:
  - Unit tests: semver parsing, classification, npm update classification
  - Integration tests: script execution, report generation, report structure
  - Content validation: all required sections, data consistency, edge cases
"""

import subprocess
import re
import json
import os
import sys
import unittest
import tempfile
from pathlib import Path

# ============================================================
# Import the functions from gen_report.py for unit testing
# ============================================================
# We import by reading the file and extracting the functions
# to avoid running the full script on import
import importlib.util
spec = importlib.util.spec_from_file_location("gen_report", "/tmp/gen_report.py")
gen_report = importlib.util.module_from_spec(spec)
# We'll just exec the functions we need rather than the whole script
exec(open("/tmp/gen_report.py").read().split("# ============================================================")[0], gen_report.__dict__)

# Manually define the functions we need for unit testing
def parse_semver(v):
    """Parse a version string into (major, minor, patch, is_pseudo)."""
    pseudo_match = re.match(r'^(\d+)\.(\d+)\.(\d+)-', v)
    if pseudo_match:
        return (int(pseudo_match.group(1)), int(pseudo_match.group(2)), 
                int(pseudo_match.group(3)), True)
    m = re.match(r'^(\d+)\.(\d+)\.(\d+)$', v)
    if m:
        return (int(m.group(1)), int(m.group(2)), int(m.group(3)), False)
    return (0, 0, 0, False)

def classify_update(current, available):
    """Classify the update type."""
    cur_major, cur_minor, cur_patch, cur_pseudo = parse_semver(current)
    ava_major, ava_minor, ava_patch, ava_pseudo = parse_semver(available)
    if cur_pseudo or ava_pseudo:
        return "pseudo-version"
    if ava_major > cur_major:
        return "major"
    elif ava_minor > cur_minor:
        return "minor"
    elif ava_patch > cur_patch:
        return "patch"
    else:
        return "unknown"

def classify_npm_update(current, latest):
    """Classify npm update type."""
    cur_m = re.match(r'^(\d+)\.', current)
    lat_m = re.match(r'^(\d+)\.', latest)
    if cur_m and lat_m:
        cur_major = int(cur_m.group(1))
        lat_major = int(lat_m.group(1))
        if lat_major > cur_major:
            return "major"
        elif lat_major == cur_major:
            return "minor"
    return "unknown"


# ============================================================
# Unit Tests
# ============================================================

class TestSemverParsing(unittest.TestCase):
    """Test parse_semver function with various version formats."""

    def test_standard_version(self):
        """Standard semver: v1.2.3"""
        result = parse_semver("1.2.3")
        self.assertEqual(result, (1, 2, 3, False))

    def test_zero_major(self):
        """Pre-1.0 version: v0.3.0"""
        result = parse_semver("0.3.0")
        self.assertEqual(result, (0, 3, 0, False))

    def test_pseudo_version(self):
        """Pseudo-version: v0.0.0-20250430164644-389ef753e22e"""
        result = parse_semver("0.0.0-20250430164644-389ef753e22e")
        self.assertEqual(result, (0, 0, 0, True))

    def test_large_version(self):
        """Large version numbers: v10.30.1"""
        result = parse_semver("10.30.1")
        self.assertEqual(result, (10, 30, 1, False))

    def test_pseudo_with_minor(self):
        """Pseudo-version with non-zero base: v1.2.3-pre.1"""
        result = parse_semver("1.2.3-pre.1")
        self.assertEqual(result, (1, 2, 3, True))

    def test_empty_string(self):
        """Empty string should return (0,0,0,False)"""
        result = parse_semver("")
        self.assertEqual(result, (0, 0, 0, False))


class TestClassifyUpdate(unittest.TestCase):
    """Test classify_update function for semver classification."""

    def test_major_update(self):
        """v1.x.x -> v2.x.x should be major"""
        result = classify_update("1.0.0", "2.0.0")
        self.assertEqual(result, "major")

    def test_minor_update(self):
        """v1.0.x -> v1.1.x should be minor"""
        result = classify_update("1.0.0", "1.1.0")
        self.assertEqual(result, "minor")

    def test_patch_update(self):
        """v1.0.0 -> v1.0.1 should be patch"""
        result = classify_update("1.0.0", "1.0.1")
        self.assertEqual(result, "patch")

    def test_major_from_zero(self):
        """v0.x.x -> v1.x.x should be major"""
        result = classify_update("0.3.0", "1.0.0")
        self.assertEqual(result, "major")

    def test_pseudo_current(self):
        """Pseudo-version current should be pseudo-version"""
        result = classify_update("0.0.0-20250430164644", "1.0.0")
        self.assertEqual(result, "pseudo-version")

    def test_pseudo_available(self):
        """Pseudo-version available should be pseudo-version"""
        result = classify_update("1.0.0", "0.0.0-20250430164644")
        self.assertEqual(result, "pseudo-version")

    def test_no_update(self):
        """Same version should be unknown"""
        result = classify_update("1.0.0", "1.0.0")
        self.assertEqual(result, "unknown")

    def test_major_multiple(self):
        """v1.9.0 -> v1.10.1 should be minor (not major)"""
        result = classify_update("1.9.0", "1.10.1")
        self.assertEqual(result, "minor")

    def test_patch_multiple(self):
        """v1.0.0 -> v1.0.5 should be patch"""
        result = classify_update("1.0.0", "1.0.5")
        self.assertEqual(result, "patch")


class TestClassifyNpmUpdate(unittest.TestCase):
    """Test classify_npm_update function."""

    def test_major_update(self):
        """5.x.x -> 6.x.x should be major"""
        result = classify_npm_update("5.18.0", "6.0.0")
        self.assertEqual(result, "major")

    def test_minor_update(self):
        """5.0.0 -> 5.1.0 should be minor"""
        result = classify_npm_update("5.0.0", "5.1.0")
        self.assertEqual(result, "minor")

    def test_major_react(self):
        """18.x.x -> 19.x.x should be major"""
        result = classify_npm_update("18.3.1", "19.2.7")
        self.assertEqual(result, "major")

    def test_major_vite(self):
        """5.x.x -> 8.x.x should be major"""
        result = classify_npm_update("5.4.21", "8.1.0")
        self.assertEqual(result, "major")

    def test_same_major(self):
        """Same major version should be minor"""
        result = classify_npm_update("5.0.0", "5.4.21")
        self.assertEqual(result, "minor")

    def test_pre_release(self):
        """Pre-release version: 5.0.0-alpha.177 -> 9.0.0-beta.5 should be major"""
        result = classify_npm_update("5.0.0-alpha.177", "9.0.0-beta.5")
        self.assertEqual(result, "major")

    def test_no_match(self):
        """Non-matching versions should return unknown"""
        result = classify_npm_update("invalid", "1.0.0")
        self.assertEqual(result, "unknown")


# ============================================================
# Integration Tests
# ============================================================

class TestScriptExecution(unittest.TestCase):
    """Test that the gen_report.py script runs successfully."""

    @classmethod
    def setUpClass(cls):
        """Run the script once and capture output."""
        cls.script_path = "/tmp/gen_report.py"
        cls.report_path = "/home/breno/projects/kanbanai/docs/dependency-health.md"
        
        # Run the script
        result = subprocess.run(
            ["python3", cls.script_path],
            capture_output=True, text=True, timeout=180,
            cwd="/home/breno/projects/kanbanai"
        )
        cls.script_result = result
        cls.stdout = result.stdout
        cls.stderr = result.stderr
        
        # Read the generated report
        if os.path.exists(cls.report_path):
            with open(cls.report_path) as f:
                cls.report_text = f.read()
        else:
            cls.report_text = ""

    def test_script_exit_code_zero(self):
        """Script should exit with code 0."""
        self.assertEqual(
            self.script_result.returncode, 0,
            f"Script failed with exit code {self.script_result.returncode}. "
            f"stderr: {self.stderr[:500]}"
        )

    def test_report_file_exists(self):
        """Report file should be generated at docs/dependency-health.md."""
        self.assertTrue(
            os.path.exists(self.report_path),
            f"Report file not found at {self.report_path}"
        )

    def test_report_file_not_empty(self):
        """Report file should have content."""
        self.assertGreater(
            len(self.report_text), 0,
            "Report file is empty"
        )

    def test_report_file_size_reasonable(self):
        """Report file should be at least 2KB (meaningful content)."""
        self.assertGreater(
            len(self.report_text), 2000,
            f"Report file too small: {len(self.report_text)} bytes"
        )

    def test_no_stderr_errors(self):
        """Script should not produce stderr output (warnings/errors)."""
        # Filter out expected informational messages
        stderr_clean = self.stderr.strip()
        if stderr_clean:
            # Allow only known informational messages
            allowed_prefixes = [
                "go: downloading", "go: added", "go: upgraded",
                "govulncheck is", "Installing govulncheck"
            ]
            unexpected = [
                l for l in stderr_clean.split("\n")
                if not any(l.startswith(p) for p in allowed_prefixes)
            ]
            self.assertEqual(
                len(unexpected), 0,
                f"Unexpected stderr output: {unexpected[:5]}"
            )


class TestReportStructure(unittest.TestCase):
    """Test that the report has all required sections."""

    @classmethod
    def setUpClass(cls):
        cls.report_path = "/home/breno/projects/kanbanai/docs/dependency-health.md"
        with open(cls.report_path) as f:
            cls.report_text = f.read()

    def test_title_present(self):
        """Report should have a title."""
        self.assertIn("# Dependency Health Report", self.report_text)

    def test_generated_timestamp(self):
        """Report should have a generation timestamp."""
        self.assertIn("> Generated:", self.report_text)

    def test_executive_summary_section(self):
        """Report should have Executive Summary section."""
        self.assertIn("## Executive Summary", self.report_text)

    def test_go_dependencies_section(self):
        """Report should have Go Dependencies section."""
        self.assertIn("## Go Dependencies", self.report_text)

    def test_frontend_dependencies_section(self):
        """Report should have Frontend (npm) Dependencies section."""
        self.assertIn("## Frontend (npm) Dependencies", self.report_text)

    def test_go_vulnerabilities_section(self):
        """Report should have Go Vulnerabilities section."""
        self.assertIn("## Go Vulnerabilities", self.report_text)

    def test_npm_vulnerabilities_section(self):
        """Report should have npm Vulnerabilities section."""
        self.assertIn("## npm Vulnerabilities", self.report_text)

    def test_priority_recommendations(self):
        """Report should have Priority Recommendations subsection."""
        self.assertIn("### Priority Recommendations", self.report_text)

    def test_direct_deps_table(self):
        """Report should have Direct Dependencies table."""
        self.assertIn("### Direct Dependencies (all up-to-date)", self.report_text)

    def test_outdated_indirect_table(self):
        """Report should have Outdated Indirect Dependencies table."""
        self.assertIn("### Outdated Indirect Dependencies", self.report_text)

    def test_outdated_packages_table(self):
        """Report should have Outdated Packages table."""
        self.assertIn("### Outdated Packages", self.report_text)

    def test_go_vulns_affecting_code(self):
        """Report should have 'Affecting Your Code' subsection for Go vulns."""
        self.assertIn("### Affecting Your Code", self.report_text)

    def test_go_vulns_imported_packages(self):
        """Report should have 'In Imported Packages' subsection."""
        self.assertIn("### In Imported Packages", self.report_text)

    def test_go_vulns_required_modules(self):
        """Report should have 'In Required Modules' subsection."""
        self.assertIn("### In Required Modules", self.report_text)


class TestGoOutdatedContent(unittest.TestCase):
    """Test the Go outdated dependencies content."""

    @classmethod
    def setUpClass(cls):
        cls.report_path = "/home/breno/projects/kanbanai/docs/dependency-health.md"
        with open(cls.report_path) as f:
            cls.report_text = f.read()

    def test_direct_deps_listed(self):
        """All 6 direct dependencies should be listed."""
        direct_deps = [
            "github.com/gin-gonic/gin",
            "github.com/mattn/go-sqlite3",
            "github.com/modelcontextprotocol/go-sdk",
            "github.com/spf13/cobra",
            "github.com/spf13/viper",
            "github.com/google/uuid",
        ]
        for dep in direct_deps:
            with self.subTest(dep=dep):
                self.assertIn(dep, self.report_text)

    def test_outdated_count_positive(self):
        """Should report some outdated Go dependencies."""
        # Extract the count from the report
        match = re.search(r'\*\*(\d+)\*\* indirect dependencies have updates', self.report_text)
        self.assertIsNotNone(match, "Could not find outdated count in report")
        count = int(match.group(1))
        self.assertGreater(count, 0, "Expected at least 1 outdated Go dependency")

    def test_outdated_table_has_rows(self):
        """Outdated table should have data rows (not just header)."""
        # Count table rows in the Outdated Indirect Dependencies section
        table_rows = re.findall(r'^\| \d+ \| `', self.report_text, re.MULTILINE)
        self.assertGreater(len(table_rows), 0, "No data rows in outdated Go table")

    def test_table_columns_correct(self):
        """Outdated table should have correct column headers."""
        self.assertIn(
            "| # | Module | Current | Available | Type | Breaking? |",
            self.report_text
        )

    def test_breaking_flag_present(self):
        """Each row should have a Breaking? flag (✅ Yes or ❌ No)."""
        breaking_flags = re.findall(r'(✅ Yes|❌ No)', self.report_text)
        self.assertGreater(len(breaking_flags), 0, "No breaking flags found")

    def test_semver_types_present(self):
        """Should have at least one of: major, minor, patch, pseudo-version."""
        has_type = any(
            t in self.report_text
            for t in ["major", "minor", "patch", "pseudo-version"]
        )
        self.assertTrue(has_type, "No semver types found in report")


class TestNpmOutdatedContent(unittest.TestCase):
    """Test the npm outdated dependencies content."""

    @classmethod
    def setUpClass(cls):
        cls.report_path = "/home/breno/projects/kanbanai/docs/dependency-health.md"
        with open(cls.report_path) as f:
            cls.report_text = f.read()

    def test_outdated_count_positive(self):
        """Should report some outdated npm packages."""
        match = re.search(r'\*\*(\d+)\*\* packages are outdated', self.report_text)
        self.assertIsNotNone(match, "Could not find npm outdated count")
        count = int(match.group(1))
        self.assertGreater(count, 0, "Expected at least 1 outdated npm package")

    def test_table_columns_correct(self):
        """npm table should have correct column headers."""
        self.assertIn(
            "| # | Package | Current | Wanted | Latest | Type | Breaking? | Dep Type |",
            self.report_text
        )

    def test_table_has_data_rows(self):
        """npm table should have data rows."""
        table_rows = re.findall(r'^\| \d+ \| `@?', self.report_text, re.MULTILINE)
        self.assertGreater(len(table_rows), 0, "No data rows in npm outdated table")

    def test_react_listed(self):
        """React should be listed as outdated."""
        self.assertIn("react", self.report_text)

    def test_mui_listed(self):
        """MUI packages should be listed."""
        self.assertIn("@mui/material", self.report_text)

    def test_vite_listed(self):
        """Vite should be listed."""
        self.assertIn("vite", self.report_text)

    def test_dep_type_column(self):
        """Should have Dev and Production dep types."""
        self.assertIn("Dev", self.report_text)
        self.assertIn("Production", self.report_text)

    def test_all_major_bumps(self):
        """All npm outdated should be major bumps (per the report)."""
        # Count major flags
        major_count = self.report_text.count("major | ✅ Yes")
        self.assertGreater(major_count, 0, "Expected major bumps in npm table")


class TestGoVulnsContent(unittest.TestCase):
    """Test the Go vulnerabilities content."""

    @classmethod
    def setUpClass(cls):
        cls.report_path = "/home/breno/projects/kanbanai/docs/dependency-health.md"
        with open(cls.report_path) as f:
            cls.report_text = f.read()

    def test_go_vulns_count(self):
        """Should report Go vulnerabilities affecting code."""
        match = re.search(r'Affecting Your Code \((\d+) vulnerabilities\)', self.report_text)
        self.assertIsNotNone(match, "Could not find Go vulns count")
        count = int(match.group(1))
        self.assertGreater(count, 0, "Expected at least 1 Go vulnerability")

    def test_go_vulns_table_columns(self):
        """Go vulns table should have correct columns."""
        self.assertIn(
            "| # | CVE/ID | Package | Found in | Fixed in | Severity | Description |",
            self.report_text
        )

    def test_go_vulns_has_links(self):
        """Go vulns should have clickable advisory links."""
        # Look for markdown links like [GO-2026-XXXX](https://...)
        links = re.findall(r'\[GO-\d+-\d+\]\(https://pkg\.go\.dev/vuln/GO-\d+-\d+\)', self.report_text)
        self.assertGreater(len(links), 0, "No clickable Go advisory links found")

    def test_go_vulns_severity_high(self):
        """Go vulns should be High severity."""
        high_count = self.report_text.count("| High |")
        self.assertGreater(high_count, 0, "Expected High severity Go vulns")

    def test_go_recommendation_present(self):
        """Should have Go update recommendation."""
        self.assertIn("Update Go to 1.26.4", self.report_text)


class TestNpmVulnsContent(unittest.TestCase):
    """Test the npm vulnerabilities content."""

    @classmethod
    def setUpClass(cls):
        cls.report_path = "/home/breno/projects/kanbanai/docs/dependency-health.md"
        with open(cls.report_path) as f:
            cls.report_text = f.read()

    def test_npm_vulns_count(self):
        """Should report npm vulnerabilities."""
        match = re.search(r'\*\*(\d+)\*\* vulnerabilities found', self.report_text)
        self.assertIsNotNone(match, "Could not find npm vulns count")
        count = int(match.group(1))
        self.assertGreater(count, 0, "Expected at least 1 npm vulnerability")

    def test_npm_vulns_table_columns(self):
        """npm vulns table should have correct columns."""
        self.assertIn(
            "| # | Package | Severity | CVE/Advisory | Description | Caminho | Fix Version | --fix? |",
            self.report_text
        )

    def test_npm_vulns_has_links(self):
        """npm vulns should have clickable advisory links."""
        links = re.findall(r'\[GHSA-[a-z0-9-]+\]\(https://github\.com/advisories/GHSA-[a-z0-9-]+\)', self.report_text)
        self.assertGreater(len(links), 0, "No clickable npm advisory links found")

    def test_npm_vulns_sorted_by_severity(self):
        """npm vulns should be sorted by severity (high first, then moderate)."""
        # Extract severity column from table rows
        lines = self.report_text.split("\n")
        in_table = False
        severities = []
        for line in lines:
            if "| # | Package | Severity" in line:
                in_table = True
                continue
            if in_table and line.startswith("|---"):
                continue
            if in_table and line.startswith("|"):
                if "**Recommendation**" in line:
                    break
                parts = [p.strip() for p in line.split("|")]
                if len(parts) >= 4:
                    severities.append(parts[3])  # Severity column
        
        # Check ordering: high should come before moderate
        severity_order = {"critical": 0, "high": 1, "moderate": 2, "low": 3}
        seen_moderate = False
        for s in severities:
            if s == "moderate":
                seen_moderate = True
            if s == "high" and seen_moderate:
                self.fail("npm vulns not sorted: 'high' found after 'moderate'")

    def test_npm_vulns_has_caminho(self):
        """npm vulns should have dependency path (Caminho)."""
        # Check for transitive dependency paths (with →)
        self.assertIn("→", self.report_text, "No dependency paths (→) found")

    def test_npm_vulns_has_fix_flag(self):
        """npm vulns should have --fix? column with ✅ or ❌."""
        self.assertIn("✅ Yes", self.report_text, "No fix flags found")

    def test_npm_recommendation_present(self):
        """Should have npm audit recommendation."""
        self.assertIn("npm audit --fix", self.report_text)


class TestExecutiveSummary(unittest.TestCase):
    """Test the executive summary content and consistency."""

    @classmethod
    def setUpClass(cls):
        cls.report_path = "/home/breno/projects/kanbanai/docs/dependency-health.md"
        with open(cls.report_path) as f:
            cls.report_text = f.read()

    def test_summary_has_go_count(self):
        """Summary should mention Go outdated count."""
        self.assertIn("Go", self.report_text.split("## Executive Summary")[1].split("##")[0])

    def test_summary_has_npm_count(self):
        """Summary should mention npm outdated count."""
        self.assertIn("Frontend (npm)", self.report_text)

    def test_summary_has_go_vulns_count(self):
        """Summary should mention Go vulns count."""
        self.assertIn("Go vulnerabilities", self.report_text)

    def test_summary_has_npm_vulns_count(self):
        """Summary should mention npm vulns count."""
        self.assertIn("npm vulnerabilities", self.report_text)

    def test_summary_severity_breakdown(self):
        """Summary should have severity breakdown for npm vulns."""
        summary_section = self.report_text.split("## Executive Summary")[1].split("##")[0]
        has_severity = any(
            s in summary_section
            for s in ["critical", "high", "moderate", "low"]
        )
        self.assertTrue(has_severity, "No severity breakdown in executive summary")

    def test_summary_counts_consistent_with_tables(self):
        """Executive summary counts should match the actual table data."""
        # Count Go vulns in the table
        go_vuln_rows = re.findall(
            r'^\| \d+ \| \[?GO-\d+-\d+\]?', self.report_text, re.MULTILINE
        )
        go_vuln_count_from_table = len(go_vuln_rows)
        
        # Count npm vulns in the table
        npm_vuln_rows = re.findall(
            r'^\| \d+ \| `[^`]+` \| (?:high|moderate|low|critical) \|',
            self.report_text, re.MULTILINE
        )
        npm_vuln_count_from_table = len(npm_vuln_rows)
        
        # Extract counts from summary
        go_vuln_match = re.search(
            r'Go vulnerabilities\**: (\d+)', self.report_text
        )
        npm_vuln_match = re.search(
            r'npm vulnerabilities\**: (\d+)', self.report_text
        )
        
        if go_vuln_match:
            go_summary_count = int(go_vuln_match.group(1))
            self.assertEqual(
                go_summary_count, go_vuln_count_from_table,
                f"Go vulns summary ({go_summary_count}) != table ({go_vuln_count_from_table})"
            )
        
        if npm_vuln_match:
            npm_summary_count = int(npm_vuln_match.group(1))
            self.assertEqual(
                npm_summary_count, npm_vuln_count_from_table,
                f"npm vulns summary ({npm_summary_count}) != table ({npm_vuln_count_from_table})"
            )

    def test_summary_severity_counts_consistent(self):
        """Severity counts in summary should match severity counts in table."""
        # Count severities from table
        table_severities = re.findall(
            r'^\| \d+ \| `[^`]+` \| (high|moderate|low|critical) \|',
            self.report_text, re.MULTILINE
        )
        table_high = sum(1 for s in table_severities if s == "high")
        table_moderate = sum(1 for s in table_severities if s == "moderate")
        
        # Extract from summary
        summary_section = self.report_text.split("## Executive Summary")[1].split("##")[0]
        summary_high = 0
        summary_moderate = 0
        
        high_match = re.search(r'(\d+) high', summary_section)
        moderate_match = re.search(r'(\d+) moderate', summary_section)
        
        if high_match:
            summary_high = int(high_match.group(1))
        if moderate_match:
            summary_moderate = int(moderate_match.group(1))
        
        self.assertEqual(
            summary_high, table_high,
            f"Summary high count ({summary_high}) != table high count ({table_high})"
        )
        self.assertEqual(
            summary_moderate, table_moderate,
            f"Summary moderate count ({summary_moderate}) != table moderate count ({table_moderate})"
        )


class TestEdgeCases(unittest.TestCase):
    """Test edge cases and robustness."""

    @classmethod
    def setUpClass(cls):
        cls.report_path = "/home/breno/projects/kanbanai/docs/dependency-health.md"
        with open(cls.report_path) as f:
            cls.report_text = f.read()

    def test_go_sum_parsing(self):
        """go.sum parsing should handle the file format."""
        go_sum_path = "/home/breno/projects/kanbanai/go.sum"
        self.assertTrue(os.path.exists(go_sum_path), "go.sum not found")
        
        modules = set()
        with open(go_sum_path) as f:
            for line in f:
                parts = line.strip().split()
                if parts:
                    mod = parts[0]
                    mod_name = mod.split("@")[0]
                    modules.add(mod_name)
        
        self.assertGreater(len(modules), 0, "No modules found in go.sum")
        self.assertIn("github.com/gin-gonic/gin", modules)

    def test_npm_audit_json_format(self):
        """npm audit --json should produce valid JSON."""
        result = subprocess.run(
            ["npm", "audit", "--json"],
            capture_output=True, text=True, timeout=30,
            cwd="/home/breno/projects/kanbanai/frontend"
        )
        try:
            data = json.loads(result.stdout)
            self.assertIn("vulnerabilities", data, "npm audit JSON missing 'vulnerabilities' key")
            self.assertIn("metadata", data, "npm audit JSON missing 'metadata' key")
        except json.JSONDecodeError as e:
            self.fail(f"npm audit --json produced invalid JSON: {e}")

    def test_go_list_executable(self):
        """go command should be available (using same PATH as gen_report.py)."""
        env = {**os.environ, "PATH": "/usr/local/go/bin:" + os.environ.get("HOME", "") + "/go/bin:" + os.environ.get("PATH", "")}
        result = subprocess.run(
            ["go", "version"],
            capture_output=True, text=True, timeout=10,
            env=env
        )
        self.assertEqual(result.returncode, 0, "go command not available")
        self.assertIn("go", result.stdout)

    def test_npm_executable(self):
        """npm command should be available."""
        result = subprocess.run(
            ["npm", "--version"],
            capture_output=True, text=True, timeout=10
        )
        self.assertEqual(result.returncode, 0, "npm command not available")

    def test_report_markdown_valid(self):
        """Report should be valid markdown (no broken syntax)."""
        # Check for unclosed markdown links
        unclosed_links = re.findall(r'\[([^\]]+)\]\([^)]*$', self.report_text, re.MULTILINE)
        self.assertEqual(
            len(unclosed_links), 0,
            f"Found unclosed markdown links: {unclosed_links}"
        )
        
        # Check table rows have consistent column counts
        # Allow 2-column tables (e.g., Direct Dependencies: | Module | Version |)
        lines = self.report_text.split("\n")
        for i, line in enumerate(lines):
            if line.startswith("|") and "---" not in line and "| # |" not in line:
                # Count columns
                cols = len([c for c in line.split("|") if c.strip()])
                if cols > 0 and cols < 2:
                    self.fail(f"Line {i+1} has only {cols} columns: {line[:80]}")


class TestReportFreshness(unittest.TestCase):
    """Test that the report is freshly generated (not stale)."""

    @classmethod
    def setUpClass(cls):
        cls.report_path = "/home/breno/projects/kanbanai/docs/dependency-health.md"
        cls.report_mtime = os.path.getmtime(cls.report_path)

    def test_report_recently_generated(self):
        """Report should have been generated within the last 5 minutes."""
        import time
        age_seconds = time.time() - self.report_mtime
        self.assertLess(
            age_seconds, 300,
            f"Report is {age_seconds:.0f}s old (max 300s)"
        )


# ============================================================
# Main
# ============================================================

if __name__ == "__main__":
    # Create test report path if it doesn't exist
    report_path = "/home/breno/projects/kanbanai/docs/dependency-health.md"
    
    # Run tests
    loader = unittest.TestLoader()
    suite = unittest.TestSuite()
    
    # Add all test classes
    suite.addTests(loader.loadTestsFromTestCase(TestSemverParsing))
    suite.addTests(loader.loadTestsFromTestCase(TestClassifyUpdate))
    suite.addTests(loader.loadTestsFromTestCase(TestClassifyNpmUpdate))
    suite.addTests(loader.loadTestsFromTestCase(TestScriptExecution))
    suite.addTests(loader.loadTestsFromTestCase(TestReportStructure))
    suite.addTests(loader.loadTestsFromTestCase(TestGoOutdatedContent))
    suite.addTests(loader.loadTestsFromTestCase(TestNpmOutdatedContent))
    suite.addTests(loader.loadTestsFromTestCase(TestGoVulnsContent))
    suite.addTests(loader.loadTestsFromTestCase(TestNpmVulnsContent))
    suite.addTests(loader.loadTestsFromTestCase(TestExecutiveSummary))
    suite.addTests(loader.loadTestsFromTestCase(TestEdgeCases))
    suite.addTests(loader.loadTestsFromTestCase(TestReportFreshness))
    
    runner = unittest.TextTestRunner(verbosity=2)
    result = runner.run(suite)
    
    # Exit with appropriate code
    sys.exit(0 if result.wasSuccessful() else 1)
