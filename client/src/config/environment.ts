/**
 * Environment configuration
 */
export const config = {
  // API base URL - can be overridden with environment variables in production
  apiBaseUrl: process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost:6060',

  // Fixed plan ID for testing
  testPlanId: '22222222-2222-2222-2222-222222222222',
};
