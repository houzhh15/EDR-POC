/**
 * 删除确认弹窗组件
 */
import React from 'react';
import { Modal, Typography } from 'antd';
import { ExclamationCircleOutlined } from '@ant-design/icons';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { endpointsApi } from '@/api/endpoints';
import { useEndpointsUIStore } from '@/stores/endpoints';
import { endpointKeys } from '../hooks/useEndpoints';

interface DeleteConfirmModalProps {
  onSuccess?: () => void;
}

const DeleteConfirmModal: React.FC<DeleteConfirmModalProps> = ({ onSuccess }) => {
  const queryClient = useQueryClient();
  const { deleteModalVisible, deleteTargetId, closeDeleteModal } = useEndpointsUIStore();

  const deleteMutation = useMutation({
    mutationFn: () => endpointsApi.delete(deleteTargetId!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: endpointKeys.all });
      closeDeleteModal();
      onSuccess?.();
    },
  });

  const handleOk = () => {
    deleteMutation.mutate();
  };

  return (
    <Modal
      title={<><ExclamationCircleOutlined style={{ color: '#ff4d4f' }} /> 删除确认</>}
      open={deleteModalVisible}
      onOk={handleOk}
      onCancel={closeDeleteModal}
      confirmLoading={deleteMutation.isPending}
      okText="确认删除"
      okButtonProps={{ danger: true }}
    >
      <Typography.Paragraph>
        确定要删除该终端吗？此操作不可恢复。
      </Typography.Paragraph>
    </Modal>
  );
};

export default DeleteConfirmModal;
