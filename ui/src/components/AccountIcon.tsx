import { useEffect, useRef, useState } from "react";
import { useGetUserSessionAuthSessionGet } from "../api/auth/auth";
import {
  IconSettings,
  IconLogout,
  IconUser,
  IconWebhook,
} from "@tabler/icons-react";

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
          <a href="/profile">
            <IconUser size={18} />
            <span>Profile</span>
          </a>
          <a href="/profile/webhooks">
            <IconWebhook size={18} />
            <span>Webhooks</span>
          </a>
          <a href="/profile/edit">
            <IconSettings size={18} />
            <span>Settings</span>
          </a>
          <a href="/auth/logout">
            <IconLogout size={18} />
            <span>Sign out</span>
          </a>
        </div>
      )}
    </div>
  );
}
