import { Card, Skeleton } from 'antd';

export default function Loading() {
  return (
    <Card style={{ margin: 0 }}>
      <Skeleton active paragraph={{ rows: 8 }} />
    </Card>
  );
}
