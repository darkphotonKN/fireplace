import Todo from '@/components/Todo';

// Todo takes no planId prop (it reads the plan from the URL), so the planId this
// page used to pass was ignored at runtime. Dropped so the build type-checks.
export default function PlanPage() {
  return <Todo />;
}
