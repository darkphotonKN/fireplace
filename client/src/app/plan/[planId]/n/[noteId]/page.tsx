"use client";

import { useParams } from "next/navigation";
import NotePage from "@/components/NotePage";

/**
 * A note's page, in the plan's middle card (FS-PGG2N R16). The plan layout
 * keeps the rest of the plan around it, as it does for a milestone's page.
 */
export default function NoteRoute() {
  const { planId, noteId } = useParams<{ planId: string; noteId: string }>();
  return <NotePage planId={planId} noteId={noteId} />;
}
