<?php

namespace App\Http\Requests;

use Illuminate\Foundation\Http\FormRequest;

class ProjectRequest extends FormRequest
{
    /**
     * Determine if the user is authorized to make this request.
     *
     * @return bool
     */
    public function authorize()
    {
        return true;
    }

    /**
     * Get the validation rules that apply to the request.
     *
     * @return array
     */
    public function rules()
    {
        return [
            'project_no' => 'unique:trn_project|max:32',
            'project_name' => 'required|max:64',
            'order_amount' => 'integer|max:2000000000|min:0',
            'from_date' => 'nullable|date',
            'to_date' => 'nullable|date',
            'details.*.process_id' => 'required|integer',
            'details.*.from_date' => 'nullable|date',
            'details.*.to_date' => 'nullable|date',
            'details.*.man_per_day' => 'nullable|integer|max:2000000000|min:1',
            'details.*.pre_cost' => 'nullable|integer|max:2000000000|min:0'
        ];
    }

    protected function prepareForValidation()
    {
        $this->merge([
            'order_amount' => preg_replace('/[￥ ,]/', '', $this->order_amount),
        ]);
        // これは正しく動かないようだ...
        for ($i = 0; $i < count($this->details); $i++ ) {
            $this->merge([
                'details.'.$i.'.pre_cost' => preg_replace('/[￥ ,]/', '', $this->details[$i]['pre_cost']),
            ]);
        }
    }
}
