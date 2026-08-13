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
          Long-term goals sit where you left them. Today&apos;s list is short on purpose — a
          few things picked this morning, and a clean slate tomorrow.
        </>
      }
      art={<ChecklistMock />}
    />
  );
}
