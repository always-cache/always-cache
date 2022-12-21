declare global {
  type Test = {
    name: string;
    id: string;
    kind?: "check" | "optimal";
  };
  type Spec = {
    name: string;
    id: string;
    tests: Test[];
  };
  type Specs = Spec[];
}

declare const specs: Specs;
export default specs;
