<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

class CreateTables extends Migration
{
    /**
     * Run the migrations.
     *
     * @return void
     */
    public function up()
    {
        Schema::create('mst_hr', function (Blueprint $table) {
            $table->string('hr_cd', 16);
            $table->string('user_name', 128);
            $table->string('name_kana', 128)->nullable();
            $table->string('password');
            $table->integer('is_admin')->default(0);
            $table->rememberToken();
            $table->string('remarks', 256)->nullable();
            $table->integer('bp_id')->nullable();
            $table->dateTime('created_at')->useCurrent();
            $table->dateTime('updated_at')->useCurrent();

            $table->primary('hr_cd');

            $table->engine = 'InnoDB';
            $table->charset = 'utf8mb4';
            $table->collation = 'utf8mb4_bin';
        });

        Schema::create('mst_bp', function (Blueprint $table) {
            $table->integer('bp_id');
            $table->string('name', 128);
            $table->string('name_kana', 256)->nullable();
            $table->string('address', 512)->nullable();
            $table->string('phone', 16)->nullable();
            $table->string('email',128)->nullable();
            $table->string('remarks', 256)->nullable();
            $table->dateTime('created_at')->useCurrent();
            $table->dateTime('updated_at')->useCurrent();

            $table->primary('bp_id');

            $table->engine = 'InnoDB';
            $table->charset = 'utf8mb4';
            $table->collation = 'utf8mb4_bin';
        });

        Schema::create('mst_department', function (Blueprint $table) {
            $table->integer('department_id');
            $table->string('name', 64);
            $table->dateTime('created_at')->useCurrent();
            $table->dateTime('updated_at')->useCurrent();

            $table->primary('department_id');

            $table->engine = 'InnoDB';
            $table->charset = 'utf8mb4';
            $table->collation = 'utf8mb4_bin';
        });

        Schema::create('mst_department_hr', function (Blueprint $table) {
            $table->integer('department_id');
            $table->string('hr_cd', 16);
            $table->dateTime('created_at')->useCurrent();
            $table->dateTime('updated_at')->useCurrent();

            $table->primary(['department_id', 'hr_cd']);

            $table->engine = 'InnoDB';
            $table->charset = 'utf8mb4';
            $table->collation = 'utf8mb4_bin';
        });

        Schema::create('mst_role', function (Blueprint $table) {
            $table->integer('role_id');
            $table->string('name', 32);
            $table->dateTime('created_at')->useCurrent();
            $table->dateTime('updated_at')->useCurrent();

            $table->primary('role_id');

            $table->engine = 'InnoDB';
            $table->charset = 'utf8mb4';
            $table->collation = 'utf8mb4_bin';
        });

        Schema::create('mst_hr_unit_price', function (Blueprint $table) {
            $table->string('hr_cd', 16);
            $table->date('from_date');
            $table->integer('role_id');
            $table->integer('price');
            $table->dateTime('created_at')->useCurrent();
            $table->dateTime('updated_at')->useCurrent();

            $table->primary(['hr_cd', 'from_date', 'role_id']);

            $table->engine = 'InnoDB';
            $table->charset = 'utf8mb4';
            $table->collation = 'utf8mb4_bin';
        });

        Schema::create('mst_process', function (Blueprint $table) {
            $table->integer('process_id');
            $table->string('name', 32);
            $table->integer('role_id')->nullable();
            $table->dateTime('created_at')->useCurrent();
            $table->dateTime('updated_at')->useCurrent();

            $table->primary('process_id');

            $table->engine = 'InnoDB';
            $table->charset = 'utf8mb4';
            $table->collation = 'utf8mb4_bin';
        });

        Schema::create('trn_project', function (Blueprint $table) {
            $table->string('project_no', 32);
            $table->string('name', 64);
            $table->integer('order_amount')->nullable();
            $table->date('from_date')->nullable();
            $table->date('to_date')->nullable();
            $table->dateTime('created_at')->useCurrent();
            $table->dateTime('updated_at')->useCurrent();

            $table->primary('project_no');

            $table->engine = 'InnoDB';
            $table->charset = 'utf8mb4';
            $table->collation = 'utf8mb4_bin';
        });

        Schema::create('trn_project_detail', function (Blueprint $table) {
            $table->string('project_no', 32);
            $table->integer('process_id');
            $table->date('from_date')->nullable();
            $table->date('to_date')->nullable();
            $table->integer('man_per_day')->nullable();
            $table->integer('pre_cost')->nullable();
            $table->dateTime('created_at')->useCurrent();
            $table->dateTime('updated_at')->useCurrent();

            $table->primary(['project_no', 'process_id']);

            $table->engine = 'InnoDB';
            $table->charset = 'utf8mb4';
            $table->collation = 'utf8mb4_bin';
        });

        Schema::create('trn_project_detail_hr', function (Blueprint $table) {
            $table->string('project_no', 32);
            $table->integer('process_id');
            $table->string('hr_cd', 16);
            $table->integer('role_id');
            $table->dateTime('created_at')->useCurrent();
            $table->dateTime('updated_at')->useCurrent();

            $table->primary(['project_no', 'process_id', 'hr_cd']);

            $table->engine = 'InnoDB';
            $table->charset = 'utf8mb4';
            $table->collation = 'utf8mb4_bin';
        });
    }

    /**
     * Reverse the migrations.
     *
     * @return void
     */
    public function down()
    {
        Schema::dropIfExists('mst_hr');
        Schema::dropIfExists('mst_bp');
        Schema::dropIfExists('mst_department');
        Schema::dropIfExists('mst_department_hr');
        Schema::dropIfExists('mst_role');
        Schema::dropIfExists('mst_hr_unit_price');
        Schema::dropIfExists('mst_process');
        Schema::dropIfExists('trn_project');
        Schema::dropIfExists('trn_project_detail');
        Schema::dropIfExists('trn_project_detail_hr');
    }
}
