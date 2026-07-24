import { buildCheckoutCancelURL } from "../billing";

describe("billing URL builders", () => {
  describe("buildCheckoutCancelURL", () => {
    it("builds checkout cancel URL with a trailing slash in origin", () => {
      const origin = "https://example.com/";
      const plan = "pro";
      expect(buildCheckoutCancelURL(origin, plan)).toBe("https://example.com/pricing?plan=pro");
    });

    it("builds checkout cancel URL without a trailing slash in origin", () => {
      const origin = "https://example.com";
      const plan = "pro";
      expect(buildCheckoutCancelURL(origin, plan)).toBe("https://example.com/pricing?plan=pro");
    });

    it("encodes the plan parameter", () => {
      const origin = "https://example.com";
      const plan = "super pro 123 !@#";
      expect(buildCheckoutCancelURL(origin, plan)).toBe(
        `https://example.com/pricing?plan=${encodeURIComponent(plan)}`
      );
    });
  });
});
