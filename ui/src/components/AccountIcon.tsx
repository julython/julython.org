import { useEffect, useRef, useState } from "react";
import { useGetUserSessionAuthSessionGet } from "../api/auth/auth";

export function AccountIcon() {
  const menuRef = useRef<HTMLDivElement>(null);
  const [isOpen, setIsOpen] = useState(false);
  const { data: session, isError } = useGetUserSessionAuthSessionGet();

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };

    if (isOpen) {
      document.addEventListener("mousedown", handleClickOutside);
    }

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [isOpen]);

  if (!session || !session.data.user || isError) {
    return (
      <a href="/auth/login/github" role="button">
        Sign in
      </a>
    );
  }

  return (
    <div className="account-menu" ref={menuRef}>
      <button onClick={() => setIsOpen(!isOpen)} className="account-button">
        <img
          src={session.data.user.avatar_url || "/images/blank_button.png"}
          alt={session.data.user.name}
        />
      </button>
      {isOpen && (
        <div className="account-dropdown">
          <div className="account-info">
            <img
              src={session.data.user.avatar_url || "/images/blank_button.png"}
              alt={session.data.user.name}
            />
            <strong>{session.data.user.name}</strong>
          </div>
          <hr />
          <a href="/settings">Settings</a>
          <a href="/auth/logout">Sign out</a>
        </div>
      )}
    </div>
  );
}
