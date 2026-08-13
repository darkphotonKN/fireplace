'use client';

import SplitSection from '../SplitSection';
import PlanCardMock from '../mocks/PlanCardMock';

export default function PlanSection() {
  return (
    <SplitSection
      heading="Every arc gets a plan."
      body={
        <>
          Not a flat list of things to do. A plan holds the whole shape of the work — what
          comes first, what waits on it, and roughly when each part lands.
        </>
      }
      art={<PlanCardMock />}
    />
  );
}
