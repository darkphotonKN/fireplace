import Todo from '@/components/Todo';

interface PageProps {
  params: {
    planId: string;
  };
}

export default function PlanPage({ params }: PageProps) {
  return <Todo planId={params.planId} />;
}
