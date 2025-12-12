from posixpath import splitext

from july.schema import FileChange

EXTENSION_MAP = {
    ".py": "Python",
    ".pyx": "Python",
    ".pxd": "Python",
    ".js": "JavaScript",
    ".ts": "TypeScript",
    ".jsx": "JavaScript",
    ".tsx": "TypeScript",
    ".rb": "Ruby",
    ".go": "Golang",
    ".rs": "Rust",
    ".java": "Java",
    ".kt": "Kotlin",
    ".scala": "Scala",
    ".c": "C/C++",
    ".cpp": "C/C++",
    ".cc": "C/C++",
    ".h": "C/C++",
    ".hpp": "C/C++",
    ".cs": "C#",
    ".fs": "F#",
    ".php": "PHP",
    ".lua": "Lua",
    ".r": "R",
    ".swift": "Swift",
    ".m": "Objective-C",
    ".clj": "Clojure",
    ".ex": "Elixir",
    ".erl": "Erlang",
    ".hs": "Haskell",
    ".ml": "OCaml",
    ".sh": "Shell",
    ".bash": "Shell",
    ".zsh": "Shell",
    ".sql": "SQL",
    ".html": "HTML/CSS",
    ".css": "HTML/CSS",
    ".scss": "HTML/CSS",
    ".sass": "HTML/CSS",
    ".md": "Documentation",
    ".rst": "Documentation",
    ".txt": "Documentation",
    ".json": "Data",
    ".yaml": "Data",
    ".yml": "Data",
    ".toml": "Data",
}


def detect_language(filename: str) -> str | None:
    _, ext = splitext(filename.lower())
    return EXTENSION_MAP.get(ext)


def parse_files(added: list, modified: list, removed: list) -> list[FileChange]:
    files = []
    for f in added:
        files.append(FileChange(file=f, type="added", language=detect_language(f)))
    for f in modified:
        files.append(FileChange(file=f, type="modified", language=detect_language(f)))
    for f in removed:
        files.append(FileChange(file=f, type="removed", language=detect_language(f)))
    return files
