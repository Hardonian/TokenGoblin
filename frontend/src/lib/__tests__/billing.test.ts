import { buildCheckoutSuccessURL, buildCheckoutCancelURL } from '../billing';

describe('billing URL builders', () => {
  describe('buildCheckoutSuccessURL', () => {
    it('should build correctly without trailing slash', () => {
      const origin = 'http://localhost:3000';
      const plan = 'pro';
      expect(buildCheckoutSuccessURL(origin, plan)).toBe('http://localhost:3000/billing?plan=pro');
    });

    it('should build correctly with trailing slash', () => {
      const origin = 'http://localhost:3000/';
      const plan = 'pro';
      expect(buildCheckoutSuccessURL(origin, plan)).toBe('http://localhost:3000/billing?plan=pro');
    });

    it('should URL encode the plan name', () => {
      const origin = 'http://localhost:3000';
      const plan = 'pro+plus';
      expect(buildCheckoutSuccessURL(origin, plan)).toBe('http://localhost:3000/billing?plan=pro%2Bplus');
    });
  });

  describe('buildCheckoutCancelURL', () => {
    it('should build correctly without trailing slash', () => {
      const origin = 'http://localhost:3000';
      const plan = 'pro';
      expect(buildCheckoutCancelURL(origin, plan)).toBe('http://localhost:3000/pricing?plan=pro');
    });

    it('should build correctly with trailing slash', () => {
      const origin = 'http://localhost:3000/';
      const plan = 'pro';
      expect(buildCheckoutCancelURL(origin, plan)).toBe('http://localhost:3000/pricing?plan=pro');
    });

    it('should URL encode the plan name', () => {
      const origin = 'http://localhost:3000';
      const plan = 'pro+plus';
      expect(buildCheckoutCancelURL(origin, plan)).toBe('http://localhost:3000/pricing?plan=pro%2Bplus');
    });
  });
});
