/**
 * 终端管理 UI 状态 Store
 */
import { create } from 'zustand';

interface EndpointsUIState {
  // 选择状态
  selectedIds: string[];
  setSelectedIds: (ids: string[]) => void;
  clearSelection: () => void;

  // 隔离弹窗状态
  isolateModalVisible: boolean;
  isolateTargetId: string | null;
  openIsolateModal: (id: string) => void;
  closeIsolateModal: () => void;

  // 删除确认弹窗状态
  deleteModalVisible: boolean;
  deleteTargetId: string | null;
  openDeleteModal: (id: string) => void;
  closeDeleteModal: () => void;
}

export const useEndpointsUIStore = create<EndpointsUIState>((set) => ({
  // 选择状态
  selectedIds: [],
  setSelectedIds: (ids) => set({ selectedIds: ids }),
  clearSelection: () => set({ selectedIds: [] }),

  // 隔离弹窗状态
  isolateModalVisible: false,
  isolateTargetId: null,
  openIsolateModal: (id) => set({ isolateModalVisible: true, isolateTargetId: id }),
  closeIsolateModal: () => set({ isolateModalVisible: false, isolateTargetId: null }),

  // 删除确认弹窗状态
  deleteModalVisible: false,
  deleteTargetId: null,
  openDeleteModal: (id) => set({ deleteModalVisible: true, deleteTargetId: id }),
  closeDeleteModal: () => set({ deleteModalVisible: false, deleteTargetId: null }),
}));
