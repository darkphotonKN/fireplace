'use client';

import SplitSection from '../SplitSection';
import DailyReviewMock from '../mocks/DailyReviewMock';

export default function ReviewSection() {
  return (
    <SplitSection
      heading="Close the day honestly."
      body={
        <>
          Ticking boxes isn&apos;t the same as knowing how the day went. A short review at the
          end — what moved, what didn&apos;t, and why — is what makes tomorrow&apos;s list
          worth trusting.
        </>
      }
      art={<DailyReviewMock />}
    />
  );
}
