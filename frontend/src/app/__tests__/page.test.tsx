import "@testing-library/jest-dom";
import { render, screen } from "@testing-library/react";
import CommandCenter from "../page";

// Mock fetch at the global level before React/SWR initialization
const mockFetch = jest.fn();
global.fetch = mockFetch;

// Mock the components that might use framer-motion or are causing errors
jest.mock("framer-motion", () => {
  const actual = jest.requireActual("framer-motion");
  return {
    __esModule: true,
    ...actual,
    AnimatePresence: ({ children }: any) => <>{children}</>,
    motion: {
      div: ({ children, ...props }: any) => <div {...props}>{children}</div>,
      span: ({ children, ...props }: any) => <span {...props}>{children}</span>,
    }
  };
});

// Mock components that might be problematic during testing
jest.mock("@/components/GoblinSpinner", () => ({
  GoblinSpinner: () => <div data-testid="goblin-spinner">Spinner</div>
}));
jest.mock("@/components/DemoMode", () => ({
  DemoMode: () => <div data-testid="demo-mode">Demo Mode</div>
}));
jest.mock("@/components/OnboardingTour", () => ({
  OnboardingTour: () => <div data-testid="onboarding-tour">Tour</div>
}));

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
  });

  it("renders the header correctly", async () => {
    render(<CommandCenter />);
    // Just testing that the component renders without throwing an error
    expect(screen.getByText("Chief Goblin's War Room")).toBeInTheDocument();
  });
});
