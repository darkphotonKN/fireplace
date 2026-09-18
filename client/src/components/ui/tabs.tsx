import * as React from 'react';
import { cn } from '@/lib/utils';

const TabsContext = React.createContext<{
  value: string;
  onValueChange: (value: string) => void;
}>({
  value: '',
  onValueChange: () => {},
});

interface TabsProps {
  defaultValue?: string;
  value?: string;
  onValueChange?: (value: string) => void;
  className?: string;
  children: React.ReactNode;
}

const Tabs = React.forwardRef<HTMLDivElement, TabsProps>(
  ({ defaultValue, value, onValueChange, className, children, ...props }, ref) => {
    const [internalValue, setInternalValue] = React.useState(defaultValue || '');
    const actualValue = value !== undefined ? value : internalValue;
    const actualOnValueChange = onValueChange || setInternalValue;

    return (
      <TabsContext.Provider value={{ value: actualValue, onValueChange: actualOnValueChange }}>
        <div ref={ref} className={cn('w-full', className)} {...props}>
          {children}
        </div>
      </TabsContext.Provider>
    );
  }
);
Tabs.displayName = 'Tabs';

/**
 * How the strip is drawn. `solid` is the original segmented control and stays
 * the default, so existing tab strips are untouched. `underline` drops the box
 * for words on a hairline with a coral underline under the active one.
 */
type TabsVariant = 'solid' | 'underline';

const TabsVariantContext = React.createContext<TabsVariant>('solid');

interface TabsListProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: TabsVariant;
}

const TabsList = React.forwardRef<HTMLDivElement, TabsListProps>(
  ({ className, variant = 'solid', ...props }, ref) => (
    <TabsVariantContext.Provider value={variant}>
      <div
        ref={ref}
        className={cn(
          variant === 'underline'
            ? 'inline-flex items-center justify-center border-b border-foreground/15'
            : 'inline-flex h-9 items-center justify-center rounded-lg bg-gray-800 p-1 text-gray-300',
          className
        )}
        {...props}
      />
    </TabsVariantContext.Provider>
  )
);
TabsList.displayName = 'TabsList';

interface TabsTriggerProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  value: string;
}

const TabsTrigger = React.forwardRef<HTMLButtonElement, TabsTriggerProps>(
  ({ className, value, ...props }, ref) => {
    const { value: selectedValue, onValueChange } = React.useContext(TabsContext);
    const variant = React.useContext(TabsVariantContext);
    const isSelected = selectedValue === value;

    return (
      <button
        ref={ref}
        type="button"
        role="tab"
        aria-selected={isSelected}
        data-state={isSelected ? 'active' : 'inactive'}
        className={cn(
          variant === 'underline'
            ? // The underline is an ::after hairline that grows from the left,
              // sitting on the list's own hairline (-bottom-px) so the two read
              // as one line.
              'relative inline-flex items-center justify-center whitespace-nowrap rounded-none px-1 pb-3 pt-2 text-base font-medium transition-colors focus-visible:outline-none focus-visible:text-primary disabled:pointer-events-none disabled:opacity-50 after:absolute after:inset-x-0 after:-bottom-px after:h-px after:origin-left after:bg-primary after:transition-transform after:duration-300'
            : 'inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-base font-medium ring-offset-white transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-950 focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50',
          variant === 'underline'
            ? isSelected
              ? 'text-primary after:scale-x-100'
              : 'text-foreground/40 hover:text-foreground/70 after:scale-x-0'
            : isSelected
              ? 'bg-gray-700 text-white shadow'
              : 'text-gray-400 hover:text-gray-200',
          className
        )}
        onClick={() => onValueChange(value)}
        {...props}
      />
    );
  }
);
TabsTrigger.displayName = 'TabsTrigger';

interface TabsContentProps extends React.HTMLAttributes<HTMLDivElement> {
  value: string;
}

const TabsContent = React.forwardRef<HTMLDivElement, TabsContentProps>(
  ({ className, value, ...props }, ref) => {
    const { value: selectedValue } = React.useContext(TabsContext);
    const isSelected = selectedValue === value;

    if (!isSelected) return null;

    return (
      <div
        ref={ref}
        className={cn(
          'mt-2 ring-offset-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-950 focus-visible:ring-offset-2',
          className
        )}
        {...props}
      />
    );
  }
);
TabsContent.displayName = 'TabsContent';

export { Tabs, TabsList, TabsTrigger, TabsContent };