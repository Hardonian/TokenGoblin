import "@testing-library/jest-dom";
import { render, screen, waitFor } from "@testing-library/react";
import CommandCenter from "../page";
import { SWRConfig } from "swr";
import { AuthProvider } from "@/lib/auth";

// Mock fetch at the global level before React/SWR initialization
const mockFetch = jest.fn();
global.fetch = mockFetch;

// Mock matchMedia for Recharts ResponsiveContainer
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: jest.fn().mockImplementation(query => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: jest.fn(), // Deprecated
    removeListener: jest.fn(), // Deprecated
    addEventListener: jest.fn(),
    removeEventListener: jest.fn(),
    dispatchEvent: jest.fn(),
  })),
});

jest.mock("next/navigation", () => ({
  useRouter() {
    return {
      prefetch: () => null,
      push: jest.fn(),
    };
  }
}));

describe("CommandCenter", () => {
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <AuthProvider>
      <SWRConfig value={{ provider: () => new Map(), dedupingInterval: 0, fetcher: async (url) => mockFetch(url).then((res: Response) => res.json()) }}>
        {children}
      </SWRConfig>
    </AuthProvider>
  );

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
  });

  afterEach(() => {
    localStorage.clear();
  });

  it("renders the header correctly", async () => {
    render(<CommandCenter />, { wrapper });
    expect(screen.getByText("[GOBLIN_CAVERN_OS]")).toBeInTheDocument();
    expect(screen.getByText("Chief Goblin's War Room")).toBeInTheDocument();
  });

  it("fetches data on load", async () => {
    // Set tenant ID so SWR doesn't pass null for the key
    localStorage.setItem("tg_tenant_id", "test-tenant");

    render(<CommandCenter />, { wrapper });

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalled();
    });
  });
});
