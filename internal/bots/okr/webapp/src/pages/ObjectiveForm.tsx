import {Col, DatePicker, Input, InputNumber,Radio, List, Modal, Row, Select, Switch, Tabs, TabsProps} from "antd";
import {Link} from "react-router-dom";
import React, {useState} from "react";
import {PlusCircleOutlined} from "@ant-design/icons";

const data = [
  {
    title: 'Ant Design Title 1',
    id: "a"
  },
  {
    title: 'Ant Design Title 2',
    id: "b"
  },
  {
    title: 'Ant Design Title 3',
    id: "d"
  },
  {
    title: 'Ant Design Title 4',
    id: "d"
  },
];




function ObjectiveForm() {
  const [isModalOpen, setIsModalOpen] = useState(false);

  const showModal = () => {
    setIsModalOpen(true);
  };

  const handleOk = () => {
    setIsModalOpen(false);
  };

  const handleCancel = () => {
    setIsModalOpen(false);
  };

  const  createForm1 = function () {
    setIsModalOpen(true);
  }

  const  createForm2 = function () {
    setIsModalOpen(true);
  }


  return (
    <div className="app objective-form">
      <h1>编辑目标 <Link to="/">完成</Link></h1>
      <Tabs centered defaultActiveKey="1" items={[
        {
          key: '1',
          label: `基本信息`,
          children: <>
            <div>
              <Input placeholder="目标标题" />
              <p className="title-tips">用简短的一句话描述目标，不包含量化的数字</p>
            </div>

            <h2>计划</h2>
            <div className="item">
              <Row className="row">
                <Col span={12}>开启</Col>
                <Col className="right" span={12}><Switch /></Col>
              </Row>
              <Row className="row">
                <Col span={12}>开始日期</Col>
                <Col className="right" span={12}><DatePicker /></Col>
              </Row>
              <Row className="row">
                <Col span={12}>结束日期</Col>
                <Col className="right" span={12}><DatePicker /></Col>
              </Row>
            </div>

            <h2>备忘</h2>
            <div>
              <Input.TextArea placeholder="备忘" rows={10}/>
            </div>
          </>,
        },
        {
          key: '2',
          label: `关键结果`,
          children: <>
            <List
              className="item"
              dataSource={data}
              renderItem={(item, index) => (
                <List.Item>
                  <List.Item.Meta
                    title={<>{index + 1}. {item.title}</>}
                  />
                </List.Item>
              )}
            />

            <div className="item button" onClick={createForm1}>
              <PlusCircleOutlined/> 新建关键结果
            </div>

            <Modal title="关键结果" open={isModalOpen} onOk={handleOk} onCancel={handleCancel}>
              <h3>标题</h3>
              <div>
                <Input />
              </div>

              <h3>初始值</h3>
              <div>
                <InputNumber />
              </div>

              <h3>目标值</h3>
              <div>
                <InputNumber />
              </div>

              <h3>取值方式</h3>
              <div>
                <Radio.Group options={[
                  { value: 'sum', label: '求和' },
                  { value: 'last', label: '最终值' },
                  { value: 'avg', label: '平均值' },
                  { value: 'max', label: '最大值' },
                ]} optionType="button" buttonStyle="solid" />
              </div>
              <p>
                - 求和：关键结果的当前值为所有记录值的和 <br/>
                - 最终值：关键结果的当前值为所有记录中最后记录的值 <br/>
                - 平均值：关键结果的当前值为所有记录中值的平均值 <br/>
                - 最大值：关键结果的当前值为所有记录中的最大值
              </p>

              <h3>备忘</h3>
              <div>
                <Input.TextArea placeholder="备忘" rows={10}/>
              </div>
            </Modal>
          </>,
        },
        {
          key: '3',
          label: `动机 & 可行性`,
          children: <>
            <h2>动机</h2>
            <List
              className="item"
              dataSource={data}
              renderItem={(item, index) => (
                <List.Item>
                  <List.Item.Meta
                    title={<>{index + 1}. {item.title}</>}
                  />
                </List.Item>
              )}
            />

            <h2>可行性</h2>
            <List
              className="item"
              dataSource={data}
              renderItem={(item, index) => (
                <List.Item>
                  <List.Item.Meta
                    title={<>{index + 1}. {item.title}</>}
                  />
                </List.Item>
              )}
            />

            <div className="item button" onClick={createForm1}>
              <PlusCircleOutlined/> 新建动机 & 可行性
            </div>

            <Modal title="关键结果" open={isModalOpen} onOk={handleOk} onCancel={handleCancel}>
              <div>
                <Radio.Group options={[
                  { value: 'a', label: '动机' },
                  { value: 'b', label: '可行性' },
                ]}  />
              </div>

              <h3>标题</h3>
              <div>
                <Input />
              </div>

              <h3>解释</h3>
              <div>
                <Input.TextArea placeholder="解释" rows={10}/>
              </div>
            </Modal>
          </>,
        },
      ]} />
    </div>
  )
}

export default ObjectiveForm;