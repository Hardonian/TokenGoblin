import "@testing-library/jest-dom";
import { render, screen, waitFor } from "@testing-library/react";
import { SWRConfig } from "swr";
import CommandCenter from "../page";
import { useAuth } from "@/lib/auth";

// Mock fetch at the global level before React/SWR initialization
const mockFetch = jest.fn();
global.fetch = mockFetch;

jest.mock("@/lib/auth", () => ({
  useAuth: jest.fn(),
  authFetcher: jest.fn((url) => mockFetch(url).then((res) => res.json())),
}));

describe("CommandCenter", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (useAuth as jest.Mock).mockReturnValue({
      apiKey: "test-api-key",
      tenantId: "test-tenant-id",
      isLoading: false,
    });
    mockFetch.mockResolvedValue({
      json: jest.fn().mockResolvedValue({
        maturity_score: 0,
        grade: "N/A",
        roi_multiplier: 0,
        active_agents: 0,
        failure_rate_pct: 0,
        avg_latency_ms: 0,
        waste_pct: 0,
        cost_leaks: [],
        zombie_agents: [],
        models: [],
        events: []
      }),
    });
  });

  const renderWithSWR = (ui: React.ReactElement) => {
    return render(
      <SWRConfig value={{ provider: () => new Map(), dedupingInterval: 0 }}>
        {ui}
      </SWRConfig>
    );
  };

  it("renders the header correctly", async () => {
    renderWithSWR(<CommandCenter />);
    expect(await screen.findByText("[GOBLIN_CAVERN_OS]")).toBeInTheDocument();
    expect(screen.getByText("Chief Goblin's War Room")).toBeInTheDocument();
  });

  it("fetches data on load", async () => {
    renderWithSWR(<CommandCenter />);
    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalled();
    });
  });
});
