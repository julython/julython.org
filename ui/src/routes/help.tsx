import { createFileRoute } from "@tanstack/react-router";
import { useGetUserSessionAuthSessionGet } from "../api/auth/auth";

export const Route = createFileRoute("/help")({
  component: Help,
});

function Help() {
  const { data: session } = useGetUserSessionAuthSessionGet();
  const isAuthenticated = !!session;

  return (
    <div className="help">
      <div className="container section-container no-border">
        <div className="row">
          <div className="span10 offset1">
            <h2 className="spread-the-word">Help me get started!</h2>
            {isAuthenticated ? (
              <p className="lead">
                First <a href="/profile/edit">edit your profile</a> and add all
                the email addresses you use to commit with. This is how we'll
                identify you, we will not display or share this information.
              </p>
            ) : (
              <p className="lead">
                First{" "}
                <a href="/signin" className="btn btn-mini btn-info">
                  Sign In
                </a>{" "}
                and add all the email addresses you use to commit with. This is
                how we'll identify you, we will not display or share this
                information.
              </p>
            )}
            <p>
              If you have any questions not answered below, please send an email
              to <a href="mailto:help@julython.org">help@julython.org</a>.
            </p>
          </div>
        </div>
      </div>

      <div className="container section-container no-border" id="webhook">
        <div className="row">
          <div className="span10 offset1">
            <h2 className="what-is-this">How do I add a webhook?</h2>
            <p className="lead">
              Add your project's webhook URL to GitHub, GitLab, or Bitbucket to
              track your commits.
            </p>
            {/* Add your webhook instructions here */}
          </div>
        </div>
      </div>

      <div className="container section-container no-border" id="points">
        <div className="row">
          <div className="span10 offset1">
            <h2 className="what-is-this">How are points scored?</h2>
            <p className="lead">
              Points are awarded to the committer, the project, and optionally
              to the location the user has specified in his/her profile.
            </p>
            <div className="row">
              <div className="span5">
                <h3>Commits</h3>
                <p>
                  You can score points by committing to a project that has a{" "}
                  <a href="#webhook">web hook</a> setup. Each commit is worth 1
                  point.
                </p>
              </div>
              <div className="span5">
                <h3>New Projects</h3>
                <p>
                  Everytime a new project is added via{" "}
                  <a href="#webhook">web hook</a> points are awarded. Each new
                  project is worth 10 points.
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="container section-container no-border" id="times">
        <div className="row">
          <div className="span10 offset1">
            <h2 className="what-is-this">When will my points count?</h2>
            <p className="lead">
              You should use your local time to 'start' at midnight on the first
              day of the month, and 'end' at midnight on the last day of the
              month.
            </p>
            <table className="table">
              <thead>
                <tr>
                  <th>Event</th>
                  <th>Start</th>
                  <th>End</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <th>Julython</th>
                  <td>July 1st</td>
                  <td>July 31st</td>
                </tr>
                <tr>
                  <th>J(an)ulython</th>
                  <td>January 1st</td>
                  <td>January 31st</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <div className="container section-container no-border" id="git">
        <div className="row">
          <div className="span10 offset1">
            <h2 className="what-is-this">Why don't my commits show up?</h2>
            <p className="lead">
              Be sure to edit your profile and add the email you use to commit
              with.
            </p>
            <div className="row">
              <div className="span5">
                <h3>Git Help</h3>
                <p>Check the existing setting:</p>
                <pre>git config --global user.email</pre>
                <p>Set a new value:</p>
                <pre>git config --global user.email "me@example.com"</pre>
                <p>Fix the email address used for the last commit:</p>
                <pre>
                  git commit --amend --author="Me &lt;me@example.com&gt;"
                </pre>
              </div>
              <div className="span5">
                <h3>Mercurial Help</h3>
                <p>Edit .hgrc (or Mercurial.ini on Windows):</p>
                <pre>{`[ui]
username = Julython Joe <me@example.com>`}</pre>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="container section-container no-border" id="location">
        <div className="row">
          <div className="span10 offset1">
            <h2 className="what-is-this">How do I set my location?</h2>
            {isAuthenticated ? (
              <p className="lead">
                <a href="/profile/edit">Edit your profile</a> and add your
                location.
              </p>
            ) : (
              <p className="lead">
                <a href="/signin" className="btn btn-mini btn-info">
                  Sign In
                </a>{" "}
                first.
              </p>
            )}
            <p className="lead">
              The location must be a valid 'city, state, country' location in
              the world.
            </p>
          </div>
        </div>
      </div>

      <div className="container section-container no-border" id="team">
        <div className="row">
          <div className="span10 offset1">
            <h2 className="what-is-this">How do I set my team?</h2>
            {isAuthenticated ? (
              <p className="lead">
                <a href="/profile/edit">Edit your profile</a> and add your team.
              </p>
            ) : (
              <p className="lead">
                <a href="/signin" className="btn btn-mini btn-info">
                  Sign In
                </a>{" "}
                first.
              </p>
            )}
            <p className="lead">
              The team is a free form field. We slugify the contents so 'Worker
              Bees' and 'worker bees' both become 'worker-bees'.
            </p>
            <p>
              * New teams must be approved first before they appear on the site.
            </p>
          </div>
        </div>
      </div>

      <div className="container section-container no-border" id="conduct">
        <div className="row">
          <div className="span10 offset1">
            <h2 className="what-is-this">What is the code of conduct?</h2>
            <p className="lead">
              Julython uses roughly the same code of conduct policy as{" "}
              <a href="https://us.pycon.org/2013/about/code-of-conduct/">
                PyCon
              </a>
              .
            </p>
            <p>
              Julython is dedicated to providing a harassment-free experience
              for everyone, regardless of gender, sexual orientation,
              disability, physical appearance, body size, race, or religion. We
              do not tolerate harassment of Julython participants in any form.
            </p>
            <p>
              Be kind to others. Do not insult or put down other attendees.
              Behave professionally. Remember that harassment and sexist,
              racist, or exclusionary jokes are not appropriate for Julython.
            </p>
            <p>
              Violating these rules may result in your account and all points
              associated removed.
            </p>
            <p>
              Thank you for helping make this a welcoming, friendly event for
              all.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
