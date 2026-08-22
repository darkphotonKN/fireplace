'use client';

import SplitSection from '../SplitSection';
import PlanCardMock from '../mocks/PlanCardMock';

export default function PlanSection() {
  return (
    <SplitSection
      heading="Every arc gets a plan."
      body={
        <>
          Not a flat list of things to do. A plan carries the shape of the work: what comes
          first, what sits under it, and what has a date on it.
        </>
      }
      art={<PlanCardMock />}
    />
  );
}
