import "@testing-library/jest-dom";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import PricingPage from "../page";
import React from "react";
import * as billingLib from "@/lib/billing";

const mockPush = jest.fn();
jest.mock("next/navigation", () => ({
  useRouter: () => ({
    push: mockPush,
  }),
  useSearchParams: () => ({
    get: jest.fn().mockReturnValue(null),
  }),
}));

jest.mock("@/lib/billing", () => ({
  createCheckoutSession: jest.fn(),
}));

describe("PricingPage", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    process.env.NEXT_PUBLIC_STRIPE_PRICE_PRO = "price_pro_test";
    process.env.NEXT_PUBLIC_STRIPE_PRICE_ENTERPRISE = "price_enterprise_test";
    // Clear cookies
    document.cookie = "tg_tenant_id=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;";
  });

  it("renders all three pricing tiers", () => {
    render(<PricingPage />);
    expect(screen.getByText("[SCOUT]")).toBeInTheDocument();
    expect(screen.getByText("[HOARDER]")).toBeInTheDocument();
    expect(screen.getByText("[WARLORD]")).toBeInTheDocument();

    expect(screen.getByText("$0")).toBeInTheDocument();
    expect(screen.getByText("$29")).toBeInTheDocument();
    expect(screen.getByText("$99")).toBeInTheDocument();
  });

  it("redirects to /signup when free Scout plan is selected", async () => {
    render(<PricingPage />);
    const scoutButton = screen.getByRole("button", { name: "[ Allocate Free ]" });
    fireEvent.click(scoutButton);

    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith("/signup");
    });
  });

  it("redirects to /signup?plan=pro when paid plan is selected without tenant session", async () => {
    render(<PricingPage />);
    const proButton = screen.getByRole("button", { name: "[ Init: Pro Trial ]" });
    fireEvent.click(proButton);

    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith("/signup?plan=pro");
    });
  });

  it("creates checkout session when logged in with tenant cookie", async () => {
    document.cookie = "tg_tenant_id=tenant_cookie_123; path=/;";
    (billingLib.createCheckoutSession as jest.Mock).mockResolvedValue({
      checkout_url: "https://checkout.stripe.com/c/pay/cs_test_cookie",
      session_id: "cs_test_cookie",
    });

    render(<PricingPage />);
    const proButton = screen.getByRole("button", { name: "[ Init: Pro Trial ]" });
    fireEvent.click(proButton);

    await waitFor(() => {
      expect(billingLib.createCheckoutSession).toHaveBeenCalledWith(
        expect.objectContaining({
          tenantId: "tenant_cookie_123",
          priceId: "price_pro_test",
        })
      );
    });
  });

  it("displays error when price id is missing", async () => {
    delete process.env.NEXT_PUBLIC_STRIPE_PRICE_PRO;
    document.cookie = "tg_tenant_id=tenant_cookie_123; path=/;";

    render(<PricingPage />);
    const proButton = screen.getByRole("button", { name: "[ Init: Pro Trial ]" });
    fireEvent.click(proButton);

    await waitFor(() => {
      expect(screen.getByText("[ERR] Stripe price id is not configured.")).toBeInTheDocument();
    });
  });
});
