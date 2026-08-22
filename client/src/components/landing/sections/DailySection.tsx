'use client';

import SplitSection from '../SplitSection';
import ChecklistMock from '../mocks/ChecklistMock';

export default function DailySection() {
  return (
    <SplitSection
      artSide="left"
      heading="Today is a short list."
      body={
        <>
          Long-term goals sit where you left them. Today&apos;s list is short on purpose. You
          pick a few things in the morning, and tomorrow it starts clean.
        </>
      }
      art={<ChecklistMock />}
    />
  );
}
