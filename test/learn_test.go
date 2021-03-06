// Copyright 2020 Dmitry Ermolov
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test

import (
	"bufio"
	"fmt"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestLearn(t *testing.T) {
	wb := SetupWorkbench(t)
	defer wb.Close()

	shellPidInt := wb.LaunchFakeShell()
	shellPid := strconv.Itoa(shellPidInt)

	wb.RunCodCmd("init", shellPid, "bash")

	wb.RunCodCmd("learn", "--", "binaries/cat.py", "--help")

	out := wb.RunCodCmd("api", "bash-complete", shellPid, "--", "binaries/cat.py", "-", "")
	scan := bufio.NewScanner(strings.NewReader(out))
	var lines []string
	for scan.Scan() {
		lines = append(lines, scan.Text())
	}
	require.Nil(t, scan.Err())

	require.Equal(t, []string{
		"-A",
		"--show-all",
		"-b",
		"--number-nonblank",
		"-e",
		"-E",
		"--show-ends",
		"-n",
		"--number",
		"-s",
		"--squeeze-blank",
		"-t",
		"-T",
		"--show-tabs",
		"-u",
		"-v",
		"--show-nonprinting",
		"--help",
		"--version",
	}, lines)
}

func TestLearnBroken(t *testing.T) {
	wb := SetupWorkbench(t)
	defer wb.Close()

	shellPid := strconv.Itoa(wb.LaunchFakeShell())

	wb.RunCodCmd("init", shellPid, "bash")

	out, err := wb.UncheckedRunCodCmd("learn", "--", "binaries/not-existent", "--help")
	require.Error(t, err)
	require.Contains(t, out, "no such file or directory")
}

func TestLearnUpdateShorter(t *testing.T) {
	wb := SetupWorkbench(t)
	defer wb.Close()

	shellPidInt := wb.LaunchFakeShell()
	shellPid := strconv.Itoa(shellPidInt)

	wb.RunCodCmd("init", shellPid, "bash")
	wb.RunCodCmd("learn", "--", "binaries/cat.py", "--foo", "--help")

	commands := wb.ParseCodListCommands(wb.RunCodCmd("list"))
	require.Equal(t,
		[]string{
			"binaries/cat.py --foo --help",
		},
		commands,
	)

	wb.RunCodCmd("learn", "--", "binaries/cat.py", "--help")

	commands = wb.ParseCodListCommands(wb.RunCodCmd("list"))
	require.Equal(t,
		[]string{
			"binaries/cat.py --help",
		},
		commands,
	)
}

func TestLearnFromPATH(t *testing.T) {
	wb := SetupWorkbench(t)
	defer wb.Close()

	shellPid := strconv.Itoa(wb.LaunchFakeShell())

	wb.RunCodCmd("init", shellPid, "bash")

	tmpBinDir := wb.InTmpDataPath("bin")
	err := os.Mkdir(tmpBinDir, 0755)
	require.NoError(t, err)

	binCat := filepath.Join(tmpBinDir, "cat.py")
	wb.CopyFile("binaries/cat.py", binCat)

	modifiedEnv := map[string]string{
		"PATH": fmt.Sprintf("%s:%s", tmpBinDir, os.Getenv("PATH")),
	}
	wb.RunCodCmdModifiedEnv(modifiedEnv, "learn", "--", "cat.py", "--help")

	out := wb.RunCodCmd("list")
	parsed := wb.ParseCodListMap(out)
	require.Equal(
		t,
		map[int]string{
			1: "bin/cat.py --help",
		},
		parsed,
	)
}

func TestMergeLearn(t *testing.T) {
	wb := SetupWorkbench(t)
	defer wb.Close()

	shellPid := strconv.Itoa(wb.LaunchFakeShell())

	wb.RunCodCmd("init", shellPid, "bash")

	tmpBinDir := wb.InTmpDataPath("bin")
	err := os.Mkdir(tmpBinDir, 0755)
	require.NoError(t, err)

	binFoo := filepath.Join(tmpBinDir, "foo")
	wb.CopyFile("binaries/foo_v1.py", binFoo)

	// We use `foo --help` on old version of `foo`.
	modifiedEnv := map[string]string{
		"PATH": fmt.Sprintf("%s:%s", tmpBinDir, os.Getenv("PATH")),
	}
	wb.RunCodCmdModifiedEnv(modifiedEnv, "learn", "--", "foo", "--help")

	out := wb.RunCodCmd("list")
	parsed := wb.ParseCodListMap(out)
	require.Equal(
		t,
		map[int]string{
			1: "bin/foo --help",
		},
		parsed,
	)

	// We update `foo` and use another command `foo --some-arg --help` to get new help.
	binFoo = filepath.Join(tmpBinDir, "foo")
	wb.CopyFile("binaries/foo_v2.py", binFoo)

	wb.RunCodCmdModifiedEnv(modifiedEnv, "learn", "--", "foo", "--some-arg", "--help")

	out = wb.RunCodCmd("list")
	parsed = wb.ParseCodListMap(out)
	require.Equal(
		t,
		map[int]string{
			1: "bin/foo --help",
			2: "bin/foo --some-arg --help",
		},
		parsed,
	)

	// Now we use `foo --help` again and this should merge our commands.
	wb.RunCodCmdModifiedEnv(modifiedEnv, "learn", "--", "foo", "--help")

	out = wb.RunCodCmd("list")
	parsed = wb.ParseCodListMap(out)
	require.Equal(
		t,
		map[int]string{
			1: "bin/foo --help",
		},
		parsed,
	)
}
