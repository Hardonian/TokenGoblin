import "@testing-library/jest-dom";
import { render, screen, waitFor } from "@testing-library/react";
import { SWRConfig } from "swr";
import CommandCenter from "../page";

jest.mock("next/navigation", () => ({
  useRouter: () => ({ push: jest.fn() }),
}));

jest.mock("@/lib/auth", () => {
  const actual = jest.requireActual("@/lib/auth");
  return {
    ...actual,
    useAuth: () => ({ tenantId: "demo-tenant", isLoading: false }),
  };
});

// Mock fetch at the global level before React/SWR initialization
const mockFetch = jest.fn();
global.fetch = mockFetch;

function renderPage() {
  return render(
    <SWRConfig value={{ provider: () => new Map(), dedupingInterval: 0, shouldRetryOnError: false }}>
      <CommandCenter />
    </SWRConfig>
  );
}

describe("CommandCenter", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
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
          tier: "free",
          cost_leaks: [],
          zombie_agents: [],
          models: [],
          events: [],
        },
      }),
    });
  });

  it("renders the header correctly", async () => {
    renderPage();
    expect(screen.getByText("[GOBLIN_CAVERN_OS]")).toBeInTheDocument();
    expect(screen.getByText("Chief Goblin's War Room")).toBeInTheDocument();
    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalled();
    });
  });

  it("fetches data on load", async () => {
    renderPage();
    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalled();
    });
  });
});
