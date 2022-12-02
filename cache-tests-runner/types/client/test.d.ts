declare global {
  type Results = {
    [key: string]: true | string[];
  };
}

export const testResults: Results;
