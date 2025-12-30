import { createFileRoute, Link } from "@tanstack/react-router";

export const Route = createFileRoute("/")({
  component: Index,
});

function Index() {
  // Replace with actual data fetching
  const total = 12345;
  const game = "Julython 2024";
  const commits: any[] = [];
  const blog = {
    title: "Welcome to Julython",
    body: "",
    postedAt: new Date(),
    user: "admin",
  };

  return (
    <>
      <div className="container">
        <div className="row">
          <div className="span12">
            <h1 id="logo">Julython</h1>
          </div>
        </div>
      </div>

      <div className="offset-container">
        <div className="container">
          <div className="row">
            <div className="span4">
              <h2 className="section-header rocket-header">
                <span className="section-icon rocket-icon" />
                What is Julython?
              </h2>
              <p>
                During the month of July, we're encouraging developers of all
                skill levels to try and work on your pet project(s) just a
                little each day. It's a great excuse to contribute to the
                communities you follow, or even dive into a language for the
                first time.
              </p>
            </div>
            <div className="span4">
              <h2 className="section-header rules-header">
                <span className="section-icon rules-icon" />
                Are There Rules?
              </h2>
              <p>
                There is only one rule, to have fun! The goal is that you either
                learn something new or to help finish a project you started. If
                you share your repository or your commits with us, we will{" "}
                <Link to="/help">tally up points</Link> for each commit or new
                project you work on{" "}
                <Link to="/help">during the month of July</Link>.
              </p>
            </div>
            <div className="span4">
              <h2 className="section-header plus-one-header">
                <span className="section-icon plus-one-icon" />
                How Do I Join In?
              </h2>
              <p>
                All you need is a project to work on that isn't your regular day
                job. We recommend you choose something open, perhaps on GitHub
                or Bitbucket, so that others can see your progress. Then{" "}
                <Link to="/help">add a webhook</Link> for your repository, and
                we'll track your progress next to everyone else.
              </p>
            </div>
          </div>
        </div>
      </div>

      <div className="container section-container no-border">
        <div className="row">
          <div className="span8">
            <h2 className="spread-the-word">
              <span id="commit-total">{total.toLocaleString()}</span> commits
              during {game}!
            </h2>
            <div id="user-barchart" className="commit-chart" />
          </div>
          <div className="span4">
            <h2 className="participating">
              <a href="/live">What's Happening?</a>
            </h2>
            <ul className="message-list" id="live-messages">
              {commits.map((commit) => (
                <li key={commit.hash} className="message">
                  <div className="media">
                    <a
                      href={`/${commit.username}/`}
                      className="thumbnail pull-left"
                    >
                      <img
                        className="media-object"
                        src={commit.pictureUrl}
                        alt={commit.username}
                      />
                    </a>
                    <div className="media-body">
                      <h4 className="media-heading">
                        {commit.timestamp} —{" "}
                        <a href={commit.projectUrl}>{commit.projectName}</a>
                      </h4>
                      <p>{commit.message.substring(0, 100)}</p>
                      <p className="hash">
                        <a href={`/${commit.username}/`}>{commit.username}</a> —{" "}
                        <a href={commit.url}>{commit.hash.substring(0, 8)}</a>
                      </p>
                    </div>
                  </div>
                </li>
              ))}
            </ul>
          </div>
        </div>
      </div>

      <div className="container section-container no-border">
        <div className="row">
          <div className="span8">
            <h2 className="spread-the-word">{blog.title}</h2>
            <p>
              <em>
                Posted on {blog.postedAt.toLocaleDateString()} by {blog.user}
              </em>
            </p>
            <div
              className="post"
              dangerouslySetInnerHTML={{ __html: blog.body }}
            />
          </div>
          <div className="span4">
            <h3>Other Posts</h3>
            {/* blog roll here */}
          </div>
        </div>
      </div>
    </>
  );
}
