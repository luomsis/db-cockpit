import { useCallback, useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { CLUSTERS } from '../lib/mockData';
import { apiGet, withFallback } from '../lib/api';
import { useBreadcrumb } from '../App';
import ClusterDetailPg from './ClusterDetailPg';
import ClusterDetailOb from './ClusterDetailOb';
import type { Cluster } from '../lib/types';

/* 集群详情分发：按库型渲染不同维度的详情页（apiserver 数据，mock 兜底） */
export default function ClusterDetail() {
  const { cid } = useParams();
  const navigate = useNavigate();
  const [cluster, setCluster] = useState<Cluster | null>(() => CLUSTERS.find(x => x.id === cid) || null);

  const reload = useCallback(() => {
    if (!cid) return;
    withFallback(apiGet<Cluster>(`/api/clusters/${cid}`), () => CLUSTERS.find(x => x.id === cid) || null)
      .then(d => { if (d) setCluster(d); });
  }, [cid]);

  useEffect(() => { reload(); }, [reload]);

  useBreadcrumb(
    cluster
      ? [{ label: '首页' }, { label: '集群', hash: '#/clusters' }, { label: cluster.name }]
      : [{ label: '首页' }, { label: '集群管理' }],
  );
  if (!cluster) { navigate('/clusters', { replace: true }); return null; }
  return cluster.type === 'oceanbase'
    ? <ClusterDetailOb cluster={cluster} reload={reload} />
    : <ClusterDetailPg cluster={cluster} reload={reload} />;
}
