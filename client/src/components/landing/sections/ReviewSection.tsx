'use client';

import SplitSection from '../SplitSection';
import DailyReviewMock from '../mocks/DailyReviewMock';

export default function ReviewSection() {
  return (
    <SplitSection
      heading="Close the day honestly."
      body={
        <>
          Ticking boxes isn&apos;t the same as knowing how the day went. Write a line at the
          end about what moved and what didn&apos;t. That&apos;s what makes tomorrow&apos;s
          list worth picking up.
        </>
      }
      art={<DailyReviewMock />}
    />
  );
}
