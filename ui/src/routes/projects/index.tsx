import { createFileRoute } from "@tanstack/react-router";
import { useGetBoardsApiV1GameBoardsGet } from "../../api/game/game";

export const Route = createFileRoute("/projects/")({
  component: Projects,
});

function Projects() {
  const { data, isLoading } = useGetBoardsApiV1GameBoardsGet();

  if (isLoading)
    return (
      <div className="container section-container no-border">Loading...</div>
    );

  const boards = data?.data.data || [];

  // Split by project size
  const largeProjects = boards.filter((b) => (b.contributor_count || 0) > 100);
  const mediumProjects = boards.filter(
    (b) => (b.contributor_count || 0) > 10 && (b.contributor_count || 0) <= 100,
  );
  const smallProjects = boards.filter((b) => (b.contributor_count || 0) <= 10);

  return (
    <div className="container section-container no-border">
      <div className="row">
        <div className="span4">
          <h3>Large Projects</h3>
          {largeProjects.map((board) => (
            <div key={board.rank} className="board-item">
              <h4>
                <a href={`/projects/${board.slug}`}>
                  {board.rank}. {board.name}
                </a>
                <span className="badge badge-info pull-right">
                  {board.points || 0} points
                </span>
              </h4>
              <p className="small">{board.commit_count || 0} commits</p>
            </div>
          ))}
        </div>

        <div className="span4">
          <h3>Medium Projects</h3>
          {mediumProjects.map((board) => (
            <div key={board.rank} className="board-item">
              <h4>
                <a href={`/projects/${board.slug}`}>
                  {board.rank}. {board.name}
                </a>
                <span className="badge badge-info pull-right">
                  {board.points || 0} points
                </span>
              </h4>
              <p className="small">{board.commit_count || 0} commits</p>
            </div>
          ))}
        </div>

        <div className="span4">
          <h3>Small Projects</h3>
          {smallProjects.map((board) => (
            <div key={board.rank} className="board-item">
              <h4>
                <a href={`/projects/${board.slug}`}>
                  {board.rank}. {board.name}
                </a>
                <span className="badge badge-info pull-right">
                  {board.points || 0} points
                </span>
              </h4>
              <p className="small">{board.commit_count || 0} commits</p>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
