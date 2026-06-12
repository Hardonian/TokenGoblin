import "@testing-library/jest-dom";
import { render, screen } from "@testing-library/react";
import BillingPage from "../page";
import React from "react";

// Mock useSearchParams
jest.mock("next/navigation", () => ({
  useSearchParams: () => new URLSearchParams(),
}));

// Mock API responses
global.fetch = jest.fn();

describe("BillingPage", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: jest.fn().mockResolvedValue({ 
        ok: true, 
        data: {
          tier: "free",
          current_month_cost_usd: 0,
          usage_limit_usd: 0,
          usage_percent: 0,
          needs_upgrade: false,
          near_limit: false,
          at_limit: false,
        }
      }),
    });
  });

  it("renders the billing header", async () => {
    render(<BillingPage />);
    expect(screen.getByText("Subscription_Control")).toBeInTheDocument();
  });
});
