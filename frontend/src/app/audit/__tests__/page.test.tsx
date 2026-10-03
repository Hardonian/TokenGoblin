import "@testing-library/jest-dom";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import AuditPage from "../page";
import React from "react";

// Mock useAuth
jest.mock("@/lib/auth", () => ({
  useAuth: () => ({
    tenantId: "tenant_test",
    apiKey: "test_key",
    isLoading: false,
    login: jest.fn(),
    logout: jest.fn(),
  }),
}));

describe("AuditPage", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    global.fetch = jest.fn();
  });

  it("renders audit page hero and packages correctly", () => {
    render(<AuditPage />);

    expect(screen.getByText(/72-Hour Executive LLM Spend Audit/i)).toBeInTheDocument();
    expect(screen.getByText(/Reclaim 30% to 50% of Your LLM Spend/i)).toBeInTheDocument();

    // Packages
    expect(screen.getByText("Standard Audit")).toBeInTheDocument();
    expect(screen.getByText("$1,500")).toBeInTheDocument();
    expect(screen.getByText("Enterprise Audit")).toBeInTheDocument();
    expect(screen.getByText("$3,000")).toBeInTheDocument();

    // Guarantee
    expect(
      screen.getByText(/If We Don't Uncover At Least 3x Our Fee, It's Free/i)
    ).toBeInTheDocument();
  });

  it("submits the audit intake lead capture form", async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ ok: true, status: "accepted" }),
    });

    render(<AuditPage />);

    const emailInput = screen.getByPlaceholderText("name@company.com");
    fireEvent.change(emailInput, { target: { value: "cto@enterprise.com" } });

    const submitBtn = screen.getByRole("button", {
      name: /Submit Audit Request/i,
    });
    fireEvent.click(submitBtn);

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        "/api/contact",
        expect.objectContaining({
          method: "POST",
          body: expect.stringContaining("cto@enterprise.com"),
        })
      );
      expect(screen.getByText(/\[✓\] Audit Request Received/i)).toBeInTheDocument();
    });
  });

  it("displays error message if lead submission fails", async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: false,
      json: async () => ({
        ok: false,
        error: { message: "Contact delivery is temporarily unavailable." },
      }),
    });

    render(<AuditPage />);

    const emailInput = screen.getByPlaceholderText("name@company.com");
    fireEvent.change(emailInput, { target: { value: "lead@test.com" } });

    const submitBtn = screen.getByRole("button", {
      name: /Submit Audit Request/i,
    });
    fireEvent.click(submitBtn);

    await waitFor(() => {
      expect(
        screen.getByText(/\[ERR\] Contact delivery is temporarily unavailable\./i)
      ).toBeInTheDocument();
    });
  });
});
