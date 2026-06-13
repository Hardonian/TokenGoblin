import "@testing-library/jest-dom";
import { render, screen, waitFor } from "@testing-library/react";
import CommandCenter from "../page";
import React from "react";

// Mock fetch
global.fetch = jest.fn();

describe("CommandCenter", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (global.fetch as jest.Mock).mockResolvedValue({
      json: jest.fn().mockResolvedValue({ 
        ok: true, 
        data: {
          maturity_score: 0,
          grade: "N/A",
          roi_multiplier: 0,
          active_agents: 0,
          failure_rate_pct: 0,
          avg_latency_ms: 0,
          waste_pct: 0
        } 
      }),
    });
  });

  it("renders the header correctly", async () => {
    render(<CommandCenter />);
    expect(screen.getByText("[TG_CMD]")).toBeInTheDocument();
    expect(screen.getByText("System Overview")).toBeInTheDocument();
  });

  it("fetches data on load", async () => {
    render(<CommandCenter />);
    
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalled();


    });
  });
});
