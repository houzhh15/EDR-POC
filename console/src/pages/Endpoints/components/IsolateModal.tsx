/**
 * 网络隔离确认弹窗组件
 */
import React from 'react';
import { Modal, Form, Input, Switch, Alert } from 'antd';
import { ExclamationCircleOutlined } from '@ant-design/icons';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { endpointsApi } from '@/api/endpoints';
import { useEndpointsUIStore } from '@/stores/endpoints';
import { endpointKeys } from '../hooks/useEndpoints';

const { TextArea } = Input;

const IsolateModal: React.FC = () => {
  const [form] = Form.useForm();
  const queryClient = useQueryClient();
  const { isolateModalVisible, isolateTargetId, closeIsolateModal } = useEndpointsUIStore();

  const isolateMutation = useMutation({
    mutationFn: (data: { reason: string; keep_management: boolean }) =>
      endpointsApi.isolate(isolateTargetId!, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: endpointKeys.all });
      closeIsolateModal();
      form.resetFields();
    },
  });

  const handleOk = async () => {
    const values = await form.validateFields();
    isolateMutation.mutate(values);
  };

  return (
    <Modal
      title={<><ExclamationCircleOutlined style={{ color: '#faad14' }} /> 网络隔离确认</>}
      open={isolateModalVisible}
      onOk={handleOk}
      onCancel={closeIsolateModal}
      confirmLoading={isolateMutation.isPending}
      okText="确认隔离"
      okButtonProps={{ danger: true }}
    >
      <Alert
        message="隔离后，该终端将无法访问网络，仅保留与管理平台的通信。"
        type="warning"
        showIcon
        style={{ marginBottom: 16 }}
      />
      <Form form={form} layout="vertical" initialValues={{ keep_management: true }}>
        <Form.Item
          name="reason"
          label="隔离原因"
          rules={[{ required: true, message: '请输入隔离原因' }]}
        >
          <TextArea rows={3} placeholder="请描述隔离该终端的原因..." />
        </Form.Item>
        <Form.Item name="keep_management" valuePropName="checked" label="保留管理通道">
          <Switch />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default IsolateModal;
