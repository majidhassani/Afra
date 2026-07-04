import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { I18nProvider } from "@/shared/i18n";
import { ToastRegion } from "@/shared/ui/toast";
import { ApiError } from "@/shared/api/client";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 10_000,
      retry: (failureCount, error) => {
        // Never retry auth failures or client errors; retry network blips once.
        if (error instanceof ApiError && error.status > 0 && error.status < 500)
          return false;
        return failureCount < 2;
      },
      refetchOnWindowFocus: false,
    },
  },
});

export function Providers({ children }: { children: ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      <I18nProvider>
        {children}
        <ToastRegion />
      </I18nProvider>
    </QueryClientProvider>
  );
}
