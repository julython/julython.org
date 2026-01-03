import { createFileRoute, Link } from "@tanstack/react-router";
import { useGetLeadersApiV1GameLeadersGet } from "../../api/game/game";

export const Route = createFileRoute("/leaders/")({
  component: Leaders,
});

function Leaders() {
  const { data, isLoading } = useGetLeadersApiV1GameLeadersGet();

  if (isLoading)
    return (
      <div className="container section-container no-border">Loading...</div>
    );

  const people = data?.data.data || [];

  return (
    <div className="container section-container no-border">
      <div className="row">
        <div className="span4">
          <h2 className="spread-the-word">People</h2>
          {people.map((person) => (
            <div key={person.rank} className="leader-item">
              <h4>
                <img
                  src={person.avatar_url || "/images/blank_button.png"}
                  alt={person.name}
                  width="32"
                  height="32"
                  style={{ borderRadius: "50%", marginRight: "10px" }}
                />
                <Link
                  to={`/u/$username`}
                  params={{ username: person.username }}
                >
                  {person.rank}. {person.name}
                </Link>
                <span className="badge badge-info pull-right">
                  {person.points || 0} points
                </span>
              </h4>
            </div>
          ))}
        </div>

        <div className="span4">
          <h2 className="teaming-up">Locations</h2>
          {/* Will be populated when locations endpoint is available */}
          <p className="lead">Coming soon...</p>
        </div>

        <div className="span4">
          <h2 className="participating">Teams</h2>
          {/* Will be populated when teams endpoint is available */}
          <p className="lead">Coming soon...</p>
        </div>
      </div>
    </div>
  );
}
