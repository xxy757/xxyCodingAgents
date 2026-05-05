'use client';

import { Button, Result } from 'antd';

export default function Error({ error, reset }: { error: Error; reset: () => void }) {
  return (
    <Result
      status="error"
      title="页面出错了"
      subTitle={error.message}
      extra={
        <Button type="primary" onClick={reset}>
          重试
        </Button>
      }
    />
  );
}
