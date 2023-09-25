import React from 'react';
import './App.sass';
import {PlusOutlined} from '@ant-design/icons';
import {
  Button,
  Cascader,
  Checkbox,
  DatePicker,
  Form,
  Input,
  InputNumber,
  Radio,
  Select,
  Slider,
  Switch,
  TreeSelect,
  Upload
} from "antd";

const {RangePicker} = DatePicker;
const {TextArea} = Input;

const tailLayout = {
  wrapperCol: { offset: 8, span: 16 },
};

function ObjectiveForm() {
  return (
    <div>
      <Form
        labelCol={{span: 4}}
        wrapperCol={{span: 14}}
        layout="horizontal"
        style={{maxWidth: 600}}
      >
        <Form.Item label="Checkbox" name="disabled" valuePropName="checked">
          <Checkbox>Checkbox</Checkbox>
        </Form.Item>
        <Form.Item label="Radio">
          <Radio.Group>
            <Radio value="apple"> Apple </Radio>
            <Radio value="pear"> Pear </Radio>
          </Radio.Group>
        </Form.Item>
        <Form.Item label="Input">
          <Input/>
        </Form.Item>
        <Form.Item label="Select">
          <Select>
            <Select.Option value="demo">Demo</Select.Option>
          </Select>
        </Form.Item>
        <Form.Item label="TreeSelect">
          <TreeSelect
            treeData={[
              {title: 'Light', value: 'light', children: [{title: 'Bamboo', value: 'bamboo'}]},
            ]}
          />
        </Form.Item>
        <Form.Item label="Cascader">
          <Cascader
            options={[
              {
                value: 'zhejiang',
                label: 'Zhejiang',
                children: [
                  {
                    value: 'hangzhou',
                    label: 'Hangzhou',
                  },
                ],
              },
            ]}
          />
        </Form.Item>
        <Form.Item label="DatePicker">
          <DatePicker/>
        </Form.Item>
        <Form.Item label="RangePicker">
          <RangePicker/>
        </Form.Item>
        <Form.Item label="InputNumber">
          <InputNumber/>
        </Form.Item>
        <Form.Item label="TextArea">
          <TextArea rows={4}/>
        </Form.Item>
        <Form.Item label="Switch" valuePropName="checked">
          <Switch/>
        </Form.Item>
        <Form.Item label="Upload" valuePropName="fileList">
          <Upload action="/upload.do" listType="picture-card">
            <div>
              <PlusOutlined/>
              <div style={{marginTop: 8}}>Upload</div>
            </div>
          </Upload>
        </Form.Item>
        <Form.Item label="Button">
          <Button>Button</Button>
        </Form.Item>
        <Form.Item label="Slider">
          <Slider/>
        </Form.Item>
        <Form.Item {...tailLayout}>
          <Button type="primary" htmlType="submit">
            Submit
          </Button>
          <Button htmlType="button" >
            Reset
          </Button>
          <Button type="link" htmlType="button" >
            Fill form
          </Button>
        </Form.Item>
      </Form>
    </div>
  );
}

export default ObjectiveForm;
