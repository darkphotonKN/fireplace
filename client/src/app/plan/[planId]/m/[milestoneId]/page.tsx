"use client";

import { useParams } from "next/navigation";
import MilestonePage from "@/components/MilestonePage";

/**
 * A milestone's page, in the plan's middle card (FS-043WP R1). The plan
 * layout keeps the rest of the plan around it.
 */
export default function MilestoneRoute() {
  const { planId, milestoneId } = useParams<{
    planId: string;
    milestoneId: string;
  }>();
  return <MilestonePage planId={planId} milestoneId={milestoneId} />;
}
