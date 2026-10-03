import "@testing-library/jest-dom";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import BillingPage from "../page";
import React from "react";
import * as billingLib from "@/lib/billing";

// Mock useSearchParams
const mockGet = jest.fn();
jest.mock("next/navigation", () => ({
  useSearchParams: () => ({
    get: mockGet,
  }),
}));

// Mock useAuth
jest.mock("@/lib/auth", () => ({
  useAuth: () => ({
    tenantId: "tenant_test_123",
    apiKey: "test_api_key",
    isLoading: false,
    login: jest.fn(),
    logout: jest.fn(),
  }),
}));

// Mock createCheckoutSession
jest.mock("@/lib/billing", () => ({
  createCheckoutSession: jest.fn(),
}));

describe("BillingPage", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockGet.mockReturnValue(null);
    process.env.NEXT_PUBLIC_STRIPE_PRICE_PRO = "price_pro_123";
    process.env.NEXT_PUBLIC_STRIPE_PRICE_ENTERPRISE = "price_enterprise_456";

    global.fetch = jest.fn().mockImplementation((url: string) => {
      if (url === "/api/billing/status") {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            ok: true,
            data: {
              tenant_id: "tenant_test_123",
              tier: "free",
              current_month_cost_usd: 15.5,
              usage_limit_usd: 100,
              usage_percent: 15.5,
              needs_upgrade: false,
              near_limit: false,
              at_limit: false,
              subscription_id: "sub_mock_active",
              stripe_customer_id: "cus_mock_active",
            },
          }),
        });
      }
      if (url === "/api/billing/portal") {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            ok: true,
            data: { portal_url: "https://billing.stripe.com/p/session_123" },
          }),
        });
      }
      return Promise.reject(new Error("Unknown route"));
    });
  });

  it("renders the billing header and plan status data", async () => {
    render(<BillingPage />);
    expect(screen.getByText("Subscription_Control")).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText("free")).toBeInTheDocument();
      expect(screen.getByText("$15.50")).toBeInTheDocument();
      expect(screen.getByText("$100")).toBeInTheDocument();
      expect(screen.getByText("15.5%")).toBeInTheDocument();
    });
  });

  it("renders query param selected plan warning banner when plan param is set", async () => {
    mockGet.mockImplementation((param: string) => (param === "plan" ? "pro" : null));
    render(<BillingPage />);

    expect(
      screen.getByText("[SYS] Selected plan: pro. Awaiting confirmation.")
    ).toBeInTheDocument();
  });

  it("triggers upgrade checkout session on clicking PRO upgrade button", async () => {
    (billingLib.createCheckoutSession as jest.Mock).mockResolvedValue({
      checkout_url: "https://checkout.stripe.com/c/pay/cs_test_123",
      session_id: "cs_test_123",
    });

    render(<BillingPage />);

    const proButton = await screen.findByRole("button", { name: "[ Init: PRO ]" });
    fireEvent.click(proButton);

    await waitFor(() => {
      expect(billingLib.createCheckoutSession).toHaveBeenCalledWith(
        expect.objectContaining({
          tenantId: "tenant_test_123",
          priceId: "price_pro_123",
        })
      );
    });
  });

  it("renders subscription details and triggers portal redirect", async () => {
    render(<BillingPage />);

    const portalBtn = await screen.findByRole("button", { name: "[ Open_Portal ]" });
    expect(screen.getByText("sub_mock_active")).toBeInTheDocument();
    expect(screen.getByText("cus_mock_active")).toBeInTheDocument();

    fireEvent.click(portalBtn);

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        "/api/billing/portal",
        expect.objectContaining({ method: "POST" })
      );
    });
  });

  it("renders error banner when billing status fails", async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: false,
      json: async () => ({
        ok: false,
        error: { message: "Failed to connect to billing database" },
      }),
    });

    render(<BillingPage />);

    await waitFor(() => {
      expect(
        screen.getByText("[ERR] Failed to connect to billing database")
      ).toBeInTheDocument();
    });
  });
});
