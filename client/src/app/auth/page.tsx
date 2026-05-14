'use client';

import { useState, useEffect } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useAuth } from '@/context/AuthContext';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';

interface FormErrors {
  name?: string;
  email?: string;
  password?: string;
  confirmPassword?: string;
}

function validateEmail(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}

function validateSignIn(email: string, password: string): FormErrors {
  const errors: FormErrors = {};
  if (!email.trim()) errors.email = 'Email is required';
  else if (!validateEmail(email)) errors.email = 'Invalid email format';
  if (!password) errors.password = 'Password is required';
  return errors;
}

function validateSignUp(
  name: string,
  email: string,
  password: string,
  confirmPassword: string,
): FormErrors {
  const errors: FormErrors = {};
  if (!name.trim()) errors.name = 'Name is required';
  if (!email.trim()) errors.email = 'Email is required';
  else if (!validateEmail(email)) errors.email = 'Invalid email format';
  if (!password) errors.password = 'Password is required';
  else if (password.length < 6) errors.password = 'Password must be at least 6 characters';
  if (password !== confirmPassword) errors.confirmPassword = 'Passwords do not match';
  return errors;
}

export default function AuthPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { isAuthenticated, isLoading, signIn, signUp } = useAuth();

  const [tab, setTab] = useState(searchParams.get('tab') === 'signup' ? 'signup' : 'signin');
  const [loading, setLoading] = useState(false);
  const [apiError, setApiError] = useState('');
  const [errors, setErrors] = useState<FormErrors>({});

  // Sign In fields
  const [signInEmail, setSignInEmail] = useState('');
  const [signInPassword, setSignInPassword] = useState('');

  // Sign Up fields
  const [signUpName, setSignUpName] = useState('');
  const [signUpEmail, setSignUpEmail] = useState('');
  const [signUpPassword, setSignUpPassword] = useState('');
  const [signUpConfirm, setSignUpConfirm] = useState('');

  // Redirect if already authenticated
  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      router.replace('/');
    }
  }, [isAuthenticated, isLoading, router]);

  const handleSignIn = async (e: React.FormEvent) => {
    e.preventDefault();
    setApiError('');

    const validationErrors = validateSignIn(signInEmail, signInPassword);
    setErrors(validationErrors);
    if (Object.keys(validationErrors).length > 0) return;

    setLoading(true);
    try {
      await signIn(signInEmail, signInPassword);
      router.replace('/');
    } catch (err) {
      setApiError(err instanceof Error ? err.message : 'Sign in failed');
    } finally {
      setLoading(false);
    }
  };

  const handleSignUp = async (e: React.FormEvent) => {
    e.preventDefault();
    setApiError('');

    const validationErrors = validateSignUp(signUpName, signUpEmail, signUpPassword, signUpConfirm);
    setErrors(validationErrors);
    if (Object.keys(validationErrors).length > 0) return;

    setLoading(true);
    try {
      await signUp(signUpName, signUpEmail, signUpPassword);
      router.replace('/');
    } catch (err) {
      setApiError(err instanceof Error ? err.message : 'Sign up failed');
    } finally {
      setLoading(false);
    }
  };

  // Clear errors when switching tabs
  const handleTabChange = (value: string) => {
    setTab(value);
    setErrors({});
    setApiError('');
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-layout">
        <div className="text-base text-[#2e2e2e]/50 dark:text-[#d1cfc0]/50">Loading...</div>
      </div>
    );
  }

  if (isAuthenticated) return null;

  return (
    <div className="flex items-center justify-center min-h-screen bg-layout">
      <div className="w-full max-w-md px-6">
        <h1 className="text-3xl font-bold text-center mb-2" style={{ color: 'rgb(247, 111, 83)' }}>
          Fireplace
        </h1>
        <p className="text-center mb-8 text-base text-[#2e2e2e]/50 dark:text-[#d1cfc0]/50">
          Your learning &amp; development hub
        </p>

        <Tabs value={tab} onValueChange={handleTabChange} className="w-full">
          <TabsList className="w-full grid grid-cols-2">
            <TabsTrigger value="signin">Sign In</TabsTrigger>
            <TabsTrigger value="signup">Sign Up</TabsTrigger>
          </TabsList>

          {apiError && (
            <div className="mt-4 p-3 rounded-md bg-red-900/20 border border-red-800/30 text-red-300 text-base">
              {apiError}
            </div>
          )}

          <TabsContent value="signin">
            <form onSubmit={handleSignIn} className="space-y-5 mt-6">
              <div>
                <label className="block text-sm font-medium uppercase tracking-wider mb-1.5 text-[#2e2e2e]/40 dark:text-[#d1cfc0]/40">
                  Email
                </label>
                <Input
                  type="email"
                  value={signInEmail}
                  onChange={(e) => setSignInEmail(e.target.value)}
                  placeholder="you@example.com"
                  className="bg-[#e5e2d3] dark:bg-[#2a2a2a] border-[#d4c9b0] dark:border-[#3a3530] text-[#2e2e2e] dark:text-[#d1cfc0] placeholder:text-[#2e2e2e]/30 dark:placeholder:text-[#d1cfc0]/30 focus-visible:ring-[rgb(247,111,83)]/40"
                  disabled={loading}
                />
                {errors.email && <p className="text-red-400 text-sm mt-1">{errors.email}</p>}
              </div>

              <div>
                <label className="block text-sm font-medium uppercase tracking-wider mb-1.5 text-[#2e2e2e]/40 dark:text-[#d1cfc0]/40">
                  Password
                </label>
                <Input
                  type="password"
                  value={signInPassword}
                  onChange={(e) => setSignInPassword(e.target.value)}
                  placeholder="••••••••"
                  className="bg-[#e5e2d3] dark:bg-[#2a2a2a] border-[#d4c9b0] dark:border-[#3a3530] text-[#2e2e2e] dark:text-[#d1cfc0] placeholder:text-[#2e2e2e]/30 dark:placeholder:text-[#d1cfc0]/30 focus-visible:ring-[rgb(247,111,83)]/40"
                  disabled={loading}
                />
                {errors.password && <p className="text-red-400 text-sm mt-1">{errors.password}</p>}
              </div>

              <Button
                type="submit"
                className="w-full bg-[rgb(247,111,83)] hover:bg-[rgb(237,101,73)] text-white"
                disabled={loading}
              >
                {loading ? 'Signing in...' : 'Sign In'}
              </Button>
            </form>
          </TabsContent>

          <TabsContent value="signup">
            <form onSubmit={handleSignUp} className="space-y-5 mt-6">
              <div>
                <label className="block text-sm font-medium uppercase tracking-wider mb-1.5 text-[#2e2e2e]/40 dark:text-[#d1cfc0]/40">
                  Name
                </label>
                <Input
                  type="text"
                  value={signUpName}
                  onChange={(e) => setSignUpName(e.target.value)}
                  placeholder="Your name"
                  className="bg-[#e5e2d3] dark:bg-[#2a2a2a] border-[#d4c9b0] dark:border-[#3a3530] text-[#2e2e2e] dark:text-[#d1cfc0] placeholder:text-[#2e2e2e]/30 dark:placeholder:text-[#d1cfc0]/30 focus-visible:ring-[rgb(247,111,83)]/40"
                  disabled={loading}
                />
                {errors.name && <p className="text-red-400 text-sm mt-1">{errors.name}</p>}
              </div>

              <div>
                <label className="block text-sm font-medium uppercase tracking-wider mb-1.5 text-[#2e2e2e]/40 dark:text-[#d1cfc0]/40">
                  Email
                </label>
                <Input
                  type="email"
                  value={signUpEmail}
                  onChange={(e) => setSignUpEmail(e.target.value)}
                  placeholder="you@example.com"
                  className="bg-[#e5e2d3] dark:bg-[#2a2a2a] border-[#d4c9b0] dark:border-[#3a3530] text-[#2e2e2e] dark:text-[#d1cfc0] placeholder:text-[#2e2e2e]/30 dark:placeholder:text-[#d1cfc0]/30 focus-visible:ring-[rgb(247,111,83)]/40"
                  disabled={loading}
                />
                {errors.email && <p className="text-red-400 text-sm mt-1">{errors.email}</p>}
              </div>

              <div>
                <label className="block text-sm font-medium uppercase tracking-wider mb-1.5 text-[#2e2e2e]/40 dark:text-[#d1cfc0]/40">
                  Password
                </label>
                <Input
                  type="password"
                  value={signUpPassword}
                  onChange={(e) => setSignUpPassword(e.target.value)}
                  placeholder="Min 6 characters"
                  className="bg-[#e5e2d3] dark:bg-[#2a2a2a] border-[#d4c9b0] dark:border-[#3a3530] text-[#2e2e2e] dark:text-[#d1cfc0] placeholder:text-[#2e2e2e]/30 dark:placeholder:text-[#d1cfc0]/30 focus-visible:ring-[rgb(247,111,83)]/40"
                  disabled={loading}
                />
                {errors.password && <p className="text-red-400 text-sm mt-1">{errors.password}</p>}
              </div>

              <div>
                <label className="block text-sm font-medium uppercase tracking-wider mb-1.5 text-[#2e2e2e]/40 dark:text-[#d1cfc0]/40">
                  Confirm Password
                </label>
                <Input
                  type="password"
                  value={signUpConfirm}
                  onChange={(e) => setSignUpConfirm(e.target.value)}
                  placeholder="••••••••"
                  className="bg-[#e5e2d3] dark:bg-[#2a2a2a] border-[#d4c9b0] dark:border-[#3a3530] text-[#2e2e2e] dark:text-[#d1cfc0] placeholder:text-[#2e2e2e]/30 dark:placeholder:text-[#d1cfc0]/30 focus-visible:ring-[rgb(247,111,83)]/40"
                  disabled={loading}
                />
                {errors.confirmPassword && (
                  <p className="text-red-400 text-sm mt-1">{errors.confirmPassword}</p>
                )}
              </div>

              <Button
                type="submit"
                className="w-full bg-[rgb(247,111,83)] hover:bg-[rgb(237,101,73)] text-white"
                disabled={loading}
              >
                {loading ? 'Creating account...' : 'Sign Up'}
              </Button>
            </form>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}
