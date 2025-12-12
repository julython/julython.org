import pytest

from july.utils.languages import detect_language


@pytest.mark.parametrize(
    "filename,expected",
    [
        pytest.param("main.py", "Python", id="python"),
        pytest.param("app.js", "JavaScript", id="javascript"),
        pytest.param("lib.rs", "Rust", id="rust"),
        pytest.param("README.md", "Documentation", id="markdown"),
        pytest.param("style.css", "HTML/CSS", id="css"),
        pytest.param("Makefile", None, id="no-extension"),
        pytest.param("unknown.xyz", None, id="unknown"),
    ],
)
def test_detect_language(filename: str, expected: str | None):
    assert detect_language(filename) == expected
