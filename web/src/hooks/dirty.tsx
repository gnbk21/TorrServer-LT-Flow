import { useState, createContext, useContext, type ReactNode } from "react";
export const DirtyContext = createContext<{
  dirty: boolean;
  setDirty: (dirty: boolean) => void;
}>({ dirty: false, setDirty: () => {} });
export function DirtyProvider({ children }: { children: ReactNode }) {
  const [dirty, setDirty] = useState(false);
  return (
    <DirtyContext.Provider value={{ dirty, setDirty }}>
      {children}
    </DirtyContext.Provider>
  );
}
export const useDirty = () => useContext(DirtyContext);
