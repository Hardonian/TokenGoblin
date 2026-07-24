import { buildCheckoutSuccessURL, buildCheckoutCancelURL } from '../billing';

describe('billing library', () => {
  describe('buildCheckoutSuccessURL', () => {
    it('appends the billing path and plan query param', () => {
      expect(buildCheckoutSuccessURL('http://localhost:3000', 'pro')).toBe('http://localhost:3000/billing?plan=pro');
    });

    it('strips trailing slashes from the origin', () => {
      expect(buildCheckoutSuccessURL('https://example.com/', 'enterprise')).toBe('https://example.com/billing?plan=enterprise');
    });

    it('encodes special characters in the plan name', () => {
      expect(buildCheckoutSuccessURL('https://app.com', 'pro plus')).toBe('https://app.com/billing?plan=pro%20plus');
    });
  });

  describe('buildCheckoutCancelURL', () => {
    it('appends the pricing path and plan query param', () => {
      expect(buildCheckoutCancelURL('http://localhost:3000', 'pro')).toBe('http://localhost:3000/pricing?plan=pro');
    });

    it('strips trailing slashes from the origin', () => {
      expect(buildCheckoutCancelURL('https://example.com/', 'enterprise')).toBe('https://example.com/pricing?plan=enterprise');
    });

    it('encodes special characters in the plan name', () => {
      expect(buildCheckoutCancelURL('https://app.com', 'pro plus')).toBe('https://app.com/pricing?plan=pro%20plus');
    });
  });
});
