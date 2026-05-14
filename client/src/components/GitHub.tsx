"use client";

import { useState } from "react";

interface GitHubRepo {
  name: string;
  description: string;
  url: string;
  updatedAt: string;
  isConnected: boolean;
}

export default function GitHub() {
  const [repo, setRepo] = useState<GitHubRepo | null>({
    name: "flow-client",
    description:
      "React learning dashboard with task management and resource tracking",
    url: "https://github.com/username/flow-client",
    updatedAt: "2 hours ago",
    isConnected: true,
  });

  const [inputUrl, setInputUrl] = useState("");
  const [isConnecting, setIsConnecting] = useState(false);

  const handleConnect = (e: React.FormEvent) => {
    e.preventDefault();
    // In a real implementation, this would call the GitHub API
    setIsConnecting(true);

    // Simulate API call
    setTimeout(() => {
      // Extract repo name from URL
      const urlParts = inputUrl.split("/");
      const repoName = urlParts[urlParts.length - 1] || "repository";

      setRepo({
        name: repoName,
        description: "Connected repository",
        url: inputUrl,
        updatedAt: "just now",
        isConnected: true,
      });

      setInputUrl("");
      setIsConnecting(false);
    }, 1000);
  };

  const disconnectRepo = () => {
    setRepo(null);
  };

  return (
    <div className="space-y-4 max-w-[700px]">
      <div className="flex items-center space-x-2 mb-4">
        <GitHubIcon />
        <h3 className="text-lg font-semibold">Repository</h3>
      </div>

      {!repo ? (
        <div className="border rounded-lg p-6">
          <p className="mb-3 text-base">
            Connect your GitHub repository to track changes and progress.
          </p>
          <form onSubmit={handleConnect} className="flex flex-col space-y-3">
            <input
              type="text"
              value={inputUrl}
              onChange={(e) => setInputUrl(e.target.value)}
              placeholder="https://github.com/username/repository"
              className="p-2 text-base border rounded w-full"
              style={{ backgroundColor: "transparent" }}
            />
            <div>
              <button
                type="submit"
                disabled={!inputUrl || isConnecting}
                className="px-4 py-2 text-base rounded text-white disabled:opacity-50"
                style={{ backgroundColor: "rgb(247, 111, 83)" }}
              >
                {isConnecting ? "Connecting..." : "Connect Repository"}
              </button>
            </div>
          </form>
          <p className="mt-4 text-sm opacity-70">
            <a
              href="https://docs.github.com/en/rest"
              target="_blank"
              rel="noopener noreferrer"
              className="underline"
              style={{ color: "rgb(247, 111, 83)" }}
            >
              Learn more about GitHub API integration
            </a>
          </p>
        </div>
      ) : (
        <div className="border rounded-lg p-4 hover:shadow-sm transition-shadow">
          <div className="flex justify-between items-start">
            <div>
              <h4 className="font-medium">
                <a
                  href={repo.url}
                  className="hover:underline"
                  style={{ color: "rgb(247, 111, 83)" }}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  {repo.name}
                </a>
              </h4>
              <p className="text-sm mt-1">{repo.description}</p>
              <div className="mt-3">
                <button
                  onClick={disconnectRepo}
                  className="text-xs py-1 px-2 border rounded"
                  style={{
                    borderColor: "rgb(247, 111, 83)",
                    color: "rgb(247, 111, 83)",
                  }}
                >
                  Disconnect
                </button>
              </div>
            </div>
            <span className="text-xs opacity-70">Updated {repo.updatedAt}</span>
          </div>
        </div>
      )}
    </div>
  );
}

// GitHub Icon Component
function GitHubIcon() {
  return (
    <svg
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="currentColor"
      xmlns="http://www.w3.org/2000/svg"
      style={{ color: "rgb(247, 111, 83)" }}
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M12 0C5.37 0 0 5.37 0 12C0 17.31 3.435 21.795 8.205 23.385C8.805 23.49 9.03 23.13 9.03 22.815C9.03 22.53 9.015 21.585 9.015 20.58C6 21.135 5.22 19.845 4.98 19.17C4.845 18.825 4.26 17.76 3.75 17.475C3.33 17.25 2.73 16.695 3.735 16.68C4.68 16.665 5.355 17.55 5.58 17.91C6.66 19.725 8.385 19.215 9.075 18.9C9.18 18.12 9.495 17.595 9.84 17.295C7.17 16.995 4.38 15.96 4.38 11.37C4.38 10.065 4.845 8.985 5.61 8.145C5.49 7.845 5.07 6.615 5.73 4.965C5.73 4.965 6.735 4.65 9.03 6.195C9.99 5.925 11.01 5.79 12.03 5.79C13.05 5.79 14.07 5.925 15.03 6.195C17.325 4.635 18.33 4.965 18.33 4.965C18.99 6.615 18.57 7.845 18.45 8.145C19.215 8.985 19.68 10.05 19.68 11.37C19.68 15.975 16.875 16.995 14.205 17.295C14.64 17.67 15.015 18.39 15.015 19.515C15.015 21.12 15 22.41 15 22.815C15 23.13 15.225 23.505 15.825 23.385C18.2072 22.5807 20.2772 21.0497 21.7437 19.0074C23.2101 16.965 23.9993 14.5143 24 12C24 5.37 18.63 0 12 0Z"
      />
    </svg>
  );
}
