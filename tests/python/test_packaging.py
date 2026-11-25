"""Guards for release plumbing: versions, metadata and workflow wiring."""

import re
from pathlib import Path

import pytest

import py_excelize as pex

ROOT = Path(__file__).resolve().parents[2]
WORKFLOWS = ROOT / ".github" / "workflows"


def read(rel):
    return (ROOT / rel).read_text(encoding="utf-8")


def test_go_and_python_versions_match():
    go = re.search(r'const Version = "([^"]+)"', read("internal/capi/capi.go")).group(1)
    assert go == pex.__version__


def test_version_file_is_single_assignment():
    assert re.fullmatch(r'__version__ = "\d+\.\d+\.\d+"\n', read("src/py_excelize/_version.py"))


def test_changelog_mentions_current_version():
    assert f"## {pex.__version__}" in read("CHANGELOG.md")


@pytest.mark.parametrize("name", ["ci.yml", "release.yml"])
def test_workflow_exists_and_has_name(name):
    assert re.search(r"^name: \w+", (WORKFLOWS / name).read_text(), re.M)


def test_release_runs_on_version_tags():
    text = read(".github/workflows/release.yml")
    assert 'tags: ["v*"]' in text


@pytest.mark.parametrize(
    "needle",
    ["ubuntu-latest", "ubuntu-24.04-arm", "macos-14", "macos-15-intel", "windows-latest"],
)
def test_release_builds_every_platform(needle):
    assert needle in read(".github/workflows/release.yml")


@pytest.mark.parametrize("py", ["3.10", "3.11", "3.12", "3.13", "3.14", "3.15"])
def test_release_verifies_every_python(py):
    assert f'"{py}"' in read(".github/workflows/release.yml")


def test_release_uses_trusted_publishing():
    text = read(".github/workflows/release.yml")
    assert "id-token: write" in text
    assert "pypa/gh-action-pypi-publish" in text
    assert "PYPI_TOKEN" not in text


def test_publish_waits_for_wheels_sdist_and_verification():
    text = read(".github/workflows/release.yml")
    assert re.search(r"needs: \[wheels, sdist, verify\]", text)


def test_ci_is_reusable_by_release():
    assert "workflow_call" in read(".github/workflows/ci.yml")


def test_pyproject_metadata():
    text = read("pyproject.toml")
    assert 'name = "py-excelize"' in text
    assert 'requires-python = ">=3.10"' in text
    assert "py_excelize._version.__version__" in text


@pytest.mark.parametrize("classifier", ["Linux", "MacOS", "Windows"])
def test_pyproject_lists_operating_systems(classifier):
    assert f"Operating System ::" in read("pyproject.toml")
    assert classifier in read("pyproject.toml")


def test_cibuildwheel_runs_the_python_tests():
    text = read("pyproject.toml")
    assert "[tool.cibuildwheel]" in text
    assert "pytest {project}/tests/python" in text


def test_python_classifiers_cover_ci_matrix():
    text = read("pyproject.toml")
    for v in ("3.10", "3.11", "3.12", "3.13", "3.14", "3.15"):
        assert f"Programming Language :: Python :: {v}" in text


def test_package_is_typed():
    assert (ROOT / "src" / "py_excelize" / "py.typed").exists()


def test_manifest_ships_go_sources_for_sdist():
    text = read("MANIFEST.in")
    for needle in ("go.mod", "cmd", "internal", "*.go"):
        assert needle in text


def test_gitignore_excludes_native_artifacts():
    text = read(".gitignore")
    for ext in ("*.so", "*.dylib", "*.dll"):
        assert ext in text


def test_setup_names_platform_specific_libraries():
    text = read("setup.py")
    for name in ("pyexcelize.dll", "libpyexcelize.dylib", "libpyexcelize.so"):
        assert name in text


def test_setup_tags_wheel_as_abi_independent():
    assert '"py3", "none"' in read("setup.py")


def test_testdata_present():
    for name in ("cells", "controls", "empty", "encrypted"):
        assert (ROOT / "testdata" / f"{name}.xlsx").is_file()


def test_license_is_mit():
    assert "MIT License" in read("LICENSE")
