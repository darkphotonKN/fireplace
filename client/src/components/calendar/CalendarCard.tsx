"use client";

import { Calendar } from './Calendar';
import { Card } from '@/components/ui/card';
import { Calendar as CalendarIcon } from 'lucide-react';
import Link from 'next/link';

interface CalendarCardProps {
  planId: string;
  className?: string;
}

export function CalendarCard({ planId, className = '' }: CalendarCardProps) {
  return (
    <Card className={`backdrop-blur-sm shadow-sm border-0 ${className}`}>
      <div className="flex items-center justify-between p-6 pb-4">
        <h2 className="text-xl font-semibold flex items-center gap-2">
          <CalendarIcon className="w-5 h-5" />
          Schedule
        </h2>
        <Link
          href={`/calendar?planId=${planId}`}
          className="text-sm text-amber-400 hover:text-amber-300 transition-colors"
        >
          View Full Calendar →
        </Link>
      </div>
      <div className="px-6 pb-6">
        <Calendar planId={planId} compact={true} />
      </div>
    </Card>
  );
}