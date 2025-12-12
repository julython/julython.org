import pytest

from july.schema import parse_project_slug


@pytest.mark.parametrize(
    "url,expected",
    [
        pytest.param(
            "https://github.com/julython/julython.org",
            "gh-julython-julython_org",
            id="github",
        ),
        pytest.param(
            "https://gitlab.com/user/my-project",
            "gl-user-my-project",
            id="gitlab",
        ),
        pytest.param(
            "https://bitbucket.org/team/repo",
            "bb-team-repo",
            id="bitbucket",
        ),
        pytest.param(
            "https://github.com/rmyers/appengine-debugtoolbar",
            "gh-rmyers-appengine-debugtoolbar",
            id="github-dashes",
        ),
        pytest.param(
            "https://example.com/user/repo",
            "example-com-user-repo",
            id="unknown-host",
        ),
        pytest.param("", "", id="empty"),
    ],
)
def test_parse_project_slug(url: str, expected: str):
    assert parse_project_slug(url) == expected
