import Link from 'next/link';

/**
 * The plan pages' back link: a real link to a fixed address, never history
 * (FS-043WP R9, G13; FS-PGG2N R17). The milestone page goes back to its plan;
 * a note page to its milestone or its plan.
 */
export default function BackLink({ href, label }: { href: string; label: string }) {
  return (
    <Link
      href={href}
      className="flex w-fit items-center text-base text-muted-foreground transition-colors hover:text-foreground"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 20 20"
        fill="currentColor"
        aria-hidden
        className="mr-1 h-4 w-4 shrink-0"
      >
        <path
          fillRule="evenodd"
          d="M9.707 14.707a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414l4-4a1 1 0 011.414 1.414L7.414 9H15a1 1 0 110 2H7.414l2.293 2.293a1 1 0 010 1.414z"
          clipRule="evenodd"
        />
      </svg>
      {label}
    </Link>
  );
}
