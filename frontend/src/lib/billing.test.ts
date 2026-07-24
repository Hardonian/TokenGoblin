import { buildCheckoutCancelURL, buildCheckoutSuccessURL } from "./billing";

describe("billing utils", () => {
  describe("buildCheckoutCancelURL", () => {
    it("should build cancel URL correctly without trailing slash", () => {
      expect(buildCheckoutCancelURL("http://localhost:3000", "pro")).toBe(
        "http://localhost:3000/pricing?plan=pro"
      );
    });

    it("should remove trailing slash from origin", () => {
      expect(buildCheckoutCancelURL("http://localhost:3000/", "pro")).toBe(
        "http://localhost:3000/pricing?plan=pro"
      );
    });

    it("should encode special characters in plan name", () => {
      expect(buildCheckoutCancelURL("http://localhost:3000", "pro plan")).toBe(
        "http://localhost:3000/pricing?plan=pro%20plan"
      );
      expect(buildCheckoutCancelURL("http://localhost:3000", "pro+plus")).toBe(
        "http://localhost:3000/pricing?plan=pro%2Bplus"
      );
    });
  });

  describe("buildCheckoutSuccessURL", () => {
    it("should build success URL correctly without trailing slash", () => {
      expect(buildCheckoutSuccessURL("http://localhost:3000", "pro")).toBe(
        "http://localhost:3000/billing?plan=pro"
      );
    });

    it("should remove trailing slash from origin", () => {
      expect(buildCheckoutSuccessURL("http://localhost:3000/", "pro")).toBe(
        "http://localhost:3000/billing?plan=pro"
      );
    });

    it("should encode special characters in plan name", () => {
      expect(buildCheckoutSuccessURL("http://localhost:3000", "pro plan")).toBe(
        "http://localhost:3000/billing?plan=pro%20plan"
      );
      expect(buildCheckoutSuccessURL("http://localhost:3000", "pro+plus")).toBe(
        "http://localhost:3000/billing?plan=pro%2Bplus"
      );
    });
  });
});
