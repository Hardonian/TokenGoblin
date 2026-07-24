import "@testing-library/jest-dom";
import { render, screen, waitFor } from "@testing-library/react";
import CommandCenter from "../page";
import { SWRConfig } from "swr";
import React from "react";
import { AuthProvider } from "@/lib/auth";

// Mock next/navigation
jest.mock("next/navigation", () => ({
  useRouter: () => ({
    push: jest.fn(),
    replace: jest.fn(),
    prefetch: jest.fn(),
  }),
}));

// Mock fetch at the global level before React/SWR initialization
const mockFetch = jest.fn();
global.fetch = mockFetch;

describe("CommandCenter", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockFetch.mockResolvedValue({
      json: jest.fn().mockResolvedValue({
        ok: true,
        data: {
          maturity_score: 0,
          grade: "N/A",
          roi_multiplier: 0,
          active_agents: 0,
          failure_rate_pct: 0,
          avg_latency_ms: 0,
          waste_pct: 0,
        },
      }),
    });
    // Set a dummy tenant id so that SWR fetchers run
    jest.spyOn(Storage.prototype, "getItem").mockImplementation(() => "demo-tenant");
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it("renders the header correctly", async () => {
    render(
      <AuthProvider>
        <SWRConfig value={{ provider: () => new Map(), dedupingInterval: 0 }}>
          <CommandCenter />
        </SWRConfig>
      </AuthProvider>
    );
    expect(await screen.findByText("[GOBLIN_CAVERN_OS]")).toBeInTheDocument();
    expect(await screen.findByText("Chief Goblin's War Room")).toBeInTheDocument();
  });

  it("fetches data on load", async () => {
    render(
      <AuthProvider>
        <SWRConfig value={{ provider: () => new Map(), dedupingInterval: 0 }}>
          <CommandCenter />
        </SWRConfig>
      </AuthProvider>
    );
    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalled();
    });
  });
});
